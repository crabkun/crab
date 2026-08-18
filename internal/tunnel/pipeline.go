package tunnel

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/crabkun/crab/internal/compress"
	"github.com/crabkun/crab/internal/crypto"
	"github.com/crabkun/crab/internal/netutil"
)

// Stream is the data-plane duplex used after the control handshake.
// Write path: plaintext -> compress -> encrypt -> conn
// Read path:  conn -> decrypt -> decompress -> plaintext
type Stream struct {
	io.Reader
	w       io.WriteCloser
	rCloser io.Closer
	conn    io.Closer

	writeOnce sync.Once
	writeErr  error
	closeOnce sync.Once
	closeErr  error
}

func (s *Stream) Write(p []byte) (int, error) {
	return s.w.Write(p)
}

// CloseWrite flushes the compressor and closes the write half. It must not
// tear down the decompressor; the peer read goroutine may still be active.
func (s *Stream) CloseWrite() error {
	s.writeOnce.Do(func() {
		s.writeErr = s.w.Close()
	})
	return s.writeErr
}

func (s *Stream) Close() error {
	s.closeOnce.Do(func() {
		_ = s.CloseWrite()
		if s.rCloser != nil {
			_ = s.rCloser.Close()
		}
		if s.conn != nil {
			s.closeErr = s.conn.Close()
		}
	})
	return s.closeErr
}

func Open(conn net.Conn, portKey, encryptMethod, compressMethod string) (*Stream, error) {
	wrapped := netutil.Once(conn)

	er, ew, err := crypto.Lookup(encryptMethod)
	if err != nil {
		return nil, err
	}
	cr, cw, err := compress.Lookup(compressMethod)
	if err != nil {
		return nil, err
	}

	encW, err := ew(portKey, wrapped)
	if err != nil {
		return nil, fmt.Errorf("init encrypt failed: %w", err)
	}
	compW, err := cw(encW)
	if err != nil {
		_ = encW.Close()
		return nil, fmt.Errorf("init compress failed: %w", err)
	}

	decR, err := er(portKey, wrapped)
	if err != nil {
		_ = compW.Close()
		return nil, fmt.Errorf("init decrypt failed: %w", err)
	}
	compR, err := cr(decR)
	if err != nil {
		_ = compW.Close()
		_ = decR.Close()
		return nil, fmt.Errorf("init decompress failed: %w", err)
	}

	return &Stream{
		Reader:  compR,
		w:       compW,
		rCloser: compR,
		conn:    wrapped,
	}, nil
}
