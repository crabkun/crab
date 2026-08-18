package netutil

import (
	"context"
	"net"
	"sync"
	"time"
)

const keepAlive = 30 * time.Second

func Listen(ctx context.Context, address string) (net.Listener, error) {
	lc := net.ListenConfig{KeepAlive: keepAlive}
	return lc.Listen(ctx, "tcp", address)
}

func Dial(ctx context.Context, address string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{
		Timeout:   timeout,
		KeepAlive: keepAlive,
	}
	return d.DialContext(ctx, "tcp", address)
}

// OnceCloser makes Close idempotent so stacked reader/writer wrappers can
// all attempt to close the same connection.
type OnceCloser struct {
	c    net.Conn
	once sync.Once
	err  error
}

func Once(c net.Conn) *OnceCloser {
	return &OnceCloser{c: c}
}

func (o *OnceCloser) Read(p []byte) (int, error)  { return o.c.Read(p) }
func (o *OnceCloser) Write(p []byte) (int, error) { return o.c.Write(p) }
func (o *OnceCloser) LocalAddr() net.Addr         { return o.c.LocalAddr() }
func (o *OnceCloser) RemoteAddr() net.Addr        { return o.c.RemoteAddr() }

func (o *OnceCloser) SetDeadline(t time.Time) error {
	return o.c.SetDeadline(t)
}

func (o *OnceCloser) SetReadDeadline(t time.Time) error {
	return o.c.SetReadDeadline(t)
}

func (o *OnceCloser) SetWriteDeadline(t time.Time) error {
	return o.c.SetWriteDeadline(t)
}

func (o *OnceCloser) Close() error {
	o.once.Do(func() { o.err = o.c.Close() })
	return o.err
}

func IsClosed(err error) bool {
	if err == nil {
		return false
	}
	return errorsIsClosed(err)
}
