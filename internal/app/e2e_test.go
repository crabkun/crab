package app_test

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/crabkun/crab/internal/client"
	"github.com/crabkun/crab/internal/config"
	"github.com/crabkun/crab/internal/logx"
	"github.com/crabkun/crab/internal/master"
	"github.com/crabkun/crab/internal/server"
)

func TestTunnelPlainAndEncrypted(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		encrypt  string
		compress string
	}{
		{name: "plain-null", encrypt: "plain", compress: "null"},
		{name: "aes-s2", encrypt: "aes-128-cfb", compress: "s2"},
		{name: "aes-zstd", encrypt: "aes-128-cfb", compress: "zstd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runTunnelCase(t, tc.encrypt, tc.compress)
		})
	}
}

func runTunnelCase(t *testing.T, encrypt, compress string) {
	t.Helper()

	logger, err := logx.New("error")
	if err != nil {
		t.Fatal(err)
	}

	backendLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = backendLn.Close() })
	go serveEcho(backendLn)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	m := master.New(config.Master{
		ListenAt:  "127.0.0.1:0",
		MasterKey: "test-master-key",
	}, logger)
	go func() { _ = m.Run(ctx) }()
	waitReady(t, m.Ready(), "master")

	portKey := "e2e-port-key-" + encrypt + "-" + compress
	srv := server.New(mustServer(t, config.File{
		Mode:      config.ModeServer,
		LogLevel:  "error",
		Master:    m.Addr(),
		MasterKey: "test-master-key",
		Ports: []config.Port{{
			Mark:           "echo",
			LocalAddress:   backendLn.Addr().String(),
			PortKey:        portKey,
			EncryptMethod:  encrypt,
			CompressMethod: compress,
		}},
	}), logger)
	go func() { _ = srv.Run(ctx) }()
	waitReady(t, srv.Ready(), "server")

	cli := client.New(mustClient(t, config.File{
		Mode:     config.ModeClient,
		LogLevel: "error",
		Master:   m.Addr(),
		Ports: []config.Port{{
			Mark:           "echo",
			LocalAddress:   "127.0.0.1:0",
			PortKey:        portKey,
			EncryptMethod:  encrypt,
			CompressMethod: compress,
		}},
	}), logger)
	go func() { _ = cli.Run(ctx) }()
	waitReady(t, cli.Ready(), "client")

	addrs := cli.Addrs()
	if len(addrs) != 1 {
		t.Fatalf("client addrs = %v", addrs)
	}

	payload := []byte("ping-through-crab-" + encrypt + "-" + compress)
	conn, err := net.DialTimeout("tcp", addrs[0], 2*time.Second)
	if err != nil {
		t.Fatalf("dial client: %v", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("got %q want %q", got, payload)
	}
}

func TestMasterKeyMismatch(t *testing.T) {
	t.Parallel()
	logger, err := logx.New("error")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	m := master.New(config.Master{ListenAt: "127.0.0.1:0", MasterKey: "right"}, logger)
	go func() { _ = m.Run(ctx) }()
	waitReady(t, m.Ready(), "master")

	srv := server.New(mustServer(t, config.File{
		Mode:      config.ModeServer,
		LogLevel:  "error",
		Master:    m.Addr(),
		MasterKey: "wrong",
		Ports: []config.Port{{
			LocalAddress:   "127.0.0.1:9",
			PortKey:        "k",
			EncryptMethod:  "plain",
			CompressMethod: "null",
		}},
	}), logger)

	err = srv.Run(ctx)
	if err == nil || err.Error() != "master key mismatch" {
		t.Fatalf("err = %v, want master key mismatch", err)
	}
}

func waitReady(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatalf("%s did not become ready", name)
	}
}

func mustServer(t *testing.T, f config.File) config.Server {
	t.Helper()
	cfg, err := f.Validate()
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Server
}

func mustClient(t *testing.T, f config.File) config.Client {
	t.Helper()
	cfg, err := f.Validate()
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Client
}

func serveEcho(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			_, _ = io.Copy(c, c)
		}(conn)
	}
}
