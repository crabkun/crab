package tunnel

import (
	"io"
	"sync"
)

const bufferSize = 32 * 1024

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, bufferSize)
		return &b
	},
}

type writeCloser interface {
	CloseWrite() error
}

// Bridge copies both directions. Each side is write-closed as soon as its
// source ends; full Close waits until both copies have returned so
// compressors (especially zstd) are not shut down while still being read.
func Bridge(a, b io.ReadWriteCloser) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		copyHalf(b, a)
	}()
	go func() {
		defer wg.Done()
		copyHalf(a, b)
	}()
	wg.Wait()
	_ = a.Close()
	_ = b.Close()
}

func copyHalf(dst io.Writer, src io.Reader) {
	bufPtr := bufPool.Get().(*[]byte)
	defer bufPool.Put(bufPtr)
	_, _ = io.CopyBuffer(dst, src, *bufPtr)
	closeWrite(dst)
}

func closeWrite(w io.Writer) {
	if c, ok := w.(writeCloser); ok {
		_ = c.CloseWrite()
		return
	}
	if c, ok := w.(io.Closer); ok {
		_ = c.Close()
	}
}
