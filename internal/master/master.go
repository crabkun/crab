package master

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/crabkun/crab/internal/config"
	"github.com/crabkun/crab/internal/netutil"
	"github.com/crabkun/crab/internal/protocol"
	"github.com/crabkun/crab/internal/tunnel"
	"github.com/google/uuid"
)

type Master struct {
	cfg      config.Master
	log      *slog.Logger
	reg      *registry
	mu       sync.RWMutex
	listener net.Listener
	ready    chan struct{}
}

func New(cfg config.Master, logger *slog.Logger) *Master {
	return &Master{
		cfg:   cfg,
		log:   logger,
		reg:   newRegistry(),
		ready: make(chan struct{}),
	}
}

func (m *Master) Addr() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.listener == nil {
		return m.cfg.ListenAt
	}
	return m.listener.Addr().String()
}

func (m *Master) Ready() <-chan struct{} {
	return m.ready
}

func (m *Master) Run(ctx context.Context) error {
	ln, err := netutil.Listen(ctx, m.cfg.ListenAt)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.listener = ln
	m.mu.Unlock()
	close(m.ready)

	m.log.Info("master running...", "listen_at", ln.Addr().String())

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || netutil.IsClosed(err) {
				return nil
			}
			m.log.Error("accept new connection failed, retrying in 1s", "err", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Duration(protocol.AcceptRetryInterval) * time.Second):
			}
			continue
		}

		_ = conn.SetDeadline(time.Now().Add(time.Duration(protocol.HandshakeTimeout) * time.Second))
		go m.handleConn(ctx, conn)
	}
}

func (m *Master) handleConn(ctx context.Context, conn net.Conn) {
	keep := false
	defer func() {
		if !keep {
			_ = conn.Close()
		}
	}()

	log := m.log.With("remote_addr", conn.RemoteAddr().String())
	log.Debug("new connection coming")

	packet, err := protocol.ReadPacket(conn)
	if err != nil {
		log.Debug("read handshake failed", "err", err)
		return
	}
	command, payload, err := protocol.SplitCode(packet)
	if err != nil || len(payload) == 0 {
		return
	}

	protocol.ClearDeadline(conn)

	switch command {
	case protocol.CommandServerHandshake:
		keep = true
		m.handleServer(ctx, conn, payload)
	case protocol.CommandClientHandshake:
		keep = true
		m.connectPortKey(conn, string(payload))
	case protocol.CommandServerAcceptClientRequest:
		keep = true
		m.matchClient(conn, string(payload))
	default:
		return
	}
}

func (m *Master) handleServer(_ context.Context, raw net.Conn, masterKey []byte) {
	conn := &lockedConn{Conn: raw}
	log := m.log.With("remote_addr", raw.RemoteAddr().String(), "remote_type", "server")

	if string(masterKey) != m.cfg.MasterKey {
		_ = conn.WriteResponse(protocol.ResponseMasterKeyMismatch, nil)
		_ = raw.Close()
		return
	}
	if err := conn.WriteResponse(protocol.ResponseReady, nil); err != nil {
		_ = raw.Close()
		return
	}

	sess := &serverSession{
		conn:     conn,
		portKeys: make(map[string]struct{}),
	}
	defer func() {
		m.reg.dropServer(sess)
		_ = raw.Close()
	}()

	for {
		packet, err := protocol.ReadPacket(raw)
		if err != nil {
			if errors.Is(err, io.EOF) || netutil.IsClosed(err) {
				log.Debug("server disconnected")
			} else {
				log.Debug("read server packet failed", "err", err)
			}
			return
		}
		command, payload, err := protocol.SplitCode(packet)
		if err != nil {
			return
		}

		switch command {
		case protocol.CommandServerRegisterPortKey:
			if len(payload) == 0 {
				return
			}
			code := protocol.ResponsePortKeyRegSuccess
			if m.reg.registerPort(sess, string(payload)) {
				code = protocol.ResponsePortKeyExist
			} else {
				log.Debug("new port key register success", "port_key", string(payload))
			}
			if err := conn.WriteResponse(code, payload); err != nil {
				return
			}
		case protocol.CommandServerRejectClientRequest:
			if len(payload) < 2 {
				return
			}
			reason := payload[0]
			guid := string(payload[1:])
			pending, ok := m.reg.takePending(guid)
			if !ok {
				continue
			}
			close(pending.done)
			_ = protocol.WriteResponse(pending.conn, protocol.ResponseServerRejectClient, []byte{reason})
			_ = pending.conn.Close()
		default:
			return
		}
	}
}

func (m *Master) connectPortKey(conn net.Conn, portKey string) {
	log := m.log.With("remote_addr", conn.RemoteAddr().String(), "remote_type", "client")

	sess, ok := m.reg.lookupServer(portKey)
	if !ok {
		_ = protocol.WriteResponse(conn, protocol.ResponsePortKeyNotExist, []byte(portKey))
		_ = conn.Close()
		return
	}

	guid := uuid.NewString()
	pending := &pendingClient{
		conn: conn,
		done: make(chan struct{}),
	}
	if !m.reg.putPending(guid, pending) {
		_ = conn.Close()
		return
	}

	keep := false
	defer func() {
		if !keep {
			_ = conn.Close()
			m.reg.deletePending(guid)
		}
	}()

	if err := sess.conn.WriteResponse(protocol.ResponseNewClientComing, protocol.EncodeNewClient(portKey, guid)); err != nil {
		return
	}

	if !waitDone(pending.done, time.Duration(protocol.AcceptWaitTimeout)*time.Second) {
		if _, ok := m.reg.takePending(guid); ok {
			_ = protocol.WriteResponse(conn, protocol.ResponsePortKeyConnectTimeout, nil)
			log.Debug("waiting server callback timed out", "port_key", portKey)
		} else {
			// Accept/reject won the race; they own the connection.
			keep = true
		}
		return
	}
	// Accept or reject both close this wait; the matching goroutine owns the conn.
	keep = true
}

func (m *Master) matchClient(conn net.Conn, guid string) {
	keep := false
	defer func() {
		if !keep {
			_ = conn.Close()
		}
	}()

	pending, ok := m.reg.takePending(guid)
	if !ok {
		return
	}
	close(pending.done)

	if err := protocol.WriteResponse(pending.conn, protocol.ResponseServerAcceptClient, nil); err != nil {
		_ = pending.conn.Close()
		return
	}

	keep = true
	tunnel.Bridge(pending.conn, conn)
}
