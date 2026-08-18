package client

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
	"golang.org/x/sync/errgroup"
)

type Client struct {
	cfg    config.Client
	log    *slog.Logger
	mu     sync.Mutex
	addrs  []string
	ready  chan struct{}
	readyN int
}

func New(cfg config.Client, logger *slog.Logger) *Client {
	return &Client{
		cfg:   cfg,
		log:   logger,
		ready: make(chan struct{}),
	}
}

func (c *Client) Ready() <-chan struct{} { return c.ready }

func (c *Client) Addrs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.addrs))
	copy(out, c.addrs)
	return out
}

func (c *Client) markListening(addr string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.addrs = append(c.addrs, addr)
	c.readyN++
	if c.readyN == len(c.cfg.Ports) {
		close(c.ready)
	}
}

func (c *Client) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for i := range c.cfg.Ports {
		port := c.cfg.Ports[i]
		g.Go(func() error {
			return c.listen(ctx, port)
		})
	}
	return g.Wait()
}

func (c *Client) listen(ctx context.Context, port config.Port) error {
	log := c.log.With("port_mark", port.Mark, "listen_at", port.LocalAddress)

	ln, err := netutil.Listen(ctx, port.LocalAddress)
	if err != nil {
		return err
	}
	defer ln.Close()
	c.markListening(ln.Addr().String())

	log.Info("port running...", "listen_at", ln.Addr().String())

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
			log.Error("accept new connection failed, retrying in 1s", "err", err)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(time.Second):
			}
			continue
		}
		go c.handleConn(ctx, conn, port)
	}
}

func (c *Client) handleConn(ctx context.Context, local net.Conn, port config.Port) {
	log := c.log.With(
		"port_mark", port.Mark,
		"listen_at", port.LocalAddress,
		"remote_addr", local.RemoteAddr().String(),
		"master", c.cfg.Master,
	)
	log.Debug("new connection coming")

	keep := false
	defer func() {
		if !keep {
			_ = local.Close()
		}
	}()

	master, err := netutil.Dial(ctx, c.cfg.Master, 10*time.Second)
	if err != nil {
		log.Error("dial master failed", "err", err)
		return
	}
	defer func() {
		if !keep {
			_ = master.Close()
		}
	}()

	if err := protocol.WriteCommand(master, protocol.CommandClientHandshake, []byte(port.PortKey)); err != nil {
		log.Error("handshake failed", "err", err)
		return
	}

	for {
		packet, err := protocol.ReadPacket(master)
		if err != nil {
			if errors.Is(err, io.EOF) || netutil.IsClosed(err) {
				log.Error("master disconnected connection")
			} else {
				log.Error("read from master failed", "err", err)
			}
			return
		}
		code, payload, err := protocol.SplitCode(packet)
		if err != nil {
			return
		}

		switch code {
		case protocol.ResponseServerAcceptClient:
			stream, err := tunnel.Open(master, port.PortKey, port.EncryptMethod, port.CompressMethod)
			if err != nil {
				log.Error(err.Error())
				return
			}
			log.Debug("client match server success")
			keep = true
			tunnel.Bridge(local, stream)
			return
		case protocol.ResponseServerRejectClient:
			if len(payload) == 0 {
				return
			}
			switch payload[0] {
			case protocol.RejectPortKeyNotExist:
				log.Error("server reported that the port key is not registered at server", "port_key", port.PortKey)
			case protocol.RejectPortKeyRemoteConnectFailed:
				log.Error("server reported that connect to local address failed")
			}
			return
		case protocol.ResponsePortKeyNotExist:
			log.Error("master reported that the port key is not registered at master", "port_key", port.PortKey)
			return
		case protocol.ResponsePortKeyConnectTimeout:
			log.Error("master reported that connect to server timeout")
			return
		default:
			log.Error("master response an unsupported code", "code", int(code))
			return
		}
	}
}
