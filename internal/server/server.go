package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/crabkun/crab/internal/config"
	"github.com/crabkun/crab/internal/netutil"
	"github.com/crabkun/crab/internal/protocol"
	"github.com/crabkun/crab/internal/tunnel"
)

type Server struct {
	cfg          config.Server
	log          *slog.Logger
	mu           sync.Mutex
	regOK        int
	registered   chan struct{}
	registeredCl bool
}

func New(cfg config.Server, logger *slog.Logger) *Server {
	return &Server{
		cfg:        cfg,
		log:        logger,
		registered: make(chan struct{}),
	}
}

func (s *Server) Ready() <-chan struct{} { return s.registered }

func (s *Server) noteRegisterResult(ok bool) {
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.regOK++
	if !s.registeredCl && s.regOK >= len(s.cfg.Ports) {
		s.registeredCl = true
		close(s.registered)
	}
}

func (s *Server) Run(ctx context.Context) error {
	log := s.log.With("master", s.cfg.Master)
	for {
		err := s.session(ctx)
		if ctx.Err() != nil {
			return nil
		}
		var fatal fatalError
		if errors.As(err, &fatal) {
			return err
		}
		if err != nil {
			log.Error("disconnected from master, reconnecting in 3s", "err", err)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Duration(protocol.ReconnectInterval) * time.Second):
		}
	}
}

func (s *Server) session(ctx context.Context) error {
	log := s.log.With("master", s.cfg.Master)

	raw, err := netutil.Dial(ctx, s.cfg.Master, 10*time.Second)
	if err != nil {
		return err
	}
	defer raw.Close()

	ctrl := &control{conn: raw}

	_ = raw.SetDeadline(time.Now().Add(time.Duration(protocol.HandshakeTimeout) * time.Second))
	if err := ctrl.WriteCommand(protocol.CommandServerHandshake, []byte(s.cfg.MasterKey)); err != nil {
		return err
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		packet, err := protocol.ReadPacket(raw)
		if err != nil {
			return err
		}
		code, payload, err := protocol.SplitCode(packet)
		if err != nil {
			return err
		}

		switch code {
		case protocol.ResponseReady:
			protocol.ClearDeadline(raw)
			for _, p := range s.cfg.Ports {
				if err := ctrl.WriteCommand(protocol.CommandServerRegisterPortKey, []byte(p.PortKey)); err != nil {
					return err
				}
			}
		case protocol.ResponseMasterKeyMismatch:
			log.Error("master reported that master key mismatch")
			return fatalError("master key mismatch")
		case protocol.ResponsePortKeyExist, protocol.ResponsePortKeyRegSuccess:
			portKey := string(payload)
			fields := []any{"port_key", portKey}
			if p, ok := s.cfg.Port(portKey); ok && p.Mark != "" {
				fields = append(fields, "port_mark", p.Mark)
			}
			if code == protocol.ResponsePortKeyExist {
				log.Error("failed to register port key because already registered", fields...)
				s.noteRegisterResult(false)
			} else {
				log.Info("port key register success", fields...)
				s.noteRegisterResult(true)
			}
		case protocol.ResponseNewClientComing:
			if len(payload) < 4 {
				continue
			}
			portKey, guid, decErr := protocol.DecodeNewClient(payload)
			if decErr != nil {
				continue
			}
			portCfg, exist := s.cfg.Port(portKey)
			if !exist {
				log.Error("master return a non-existent port key", "port_key", portKey, "client_guid", guid)
				_ = ctrl.WriteCommand(protocol.CommandServerRejectClientRequest, append([]byte{protocol.RejectPortKeyNotExist}, guid...))
				continue
			}
			log.Debug("new client come from master",
				"port_key", portKey,
				"client_guid", guid,
				"port_mark", portCfg.Mark,
			)
			go s.handleClient(ctx, ctrl, portCfg, guid)
		default:
			log.Error("master response an unsupported code", "code", int(code))
		}
	}
}

func (s *Server) handleClient(ctx context.Context, ctrl *control, port config.Port, guid string) {
	log := s.log.With(
		"local_addr", port.LocalAddress,
		"master", s.cfg.Master,
		"client_guid", guid,
		"port_mark", port.Mark,
	)

	remote, err := netutil.Dial(ctx, port.LocalAddress, time.Duration(protocol.LocalDialTimeout)*time.Second)
	if err != nil {
		log.Error("connect to local address failed", "err", err)
		_ = ctrl.WriteCommand(protocol.CommandServerRejectClientRequest, append([]byte{protocol.RejectPortKeyRemoteConnectFailed}, guid...))
		return
	}

	callback, err := netutil.Dial(ctx, s.cfg.Master, 10*time.Second)
	if err != nil {
		log.Error("callback master failed", "err", err)
		_ = remote.Close()
		return
	}

	if err := protocol.WriteCommand(callback, protocol.CommandServerAcceptClientRequest, []byte(guid)); err != nil {
		log.Error("callback master failed", "err", err)
		_ = callback.Close()
		_ = remote.Close()
		return
	}

	stream, err := tunnel.Open(callback, port.PortKey, port.EncryptMethod, port.CompressMethod)
	if err != nil {
		log.Error(err.Error())
		_ = callback.Close()
		_ = remote.Close()
		return
	}

	log.Debug("client match success")
	tunnel.Bridge(remote, stream)
}

// control serializes writes on the long-lived server<->master socket.
type control struct {
	mu   sync.Mutex
	conn net.Conn
}

func (c *control) WriteCommand(command byte, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return protocol.WriteCommand(c.conn, command, data)
}

type fatalError string

func (e fatalError) Error() string { return string(e) }
