// Package compress implements the data-plane compressors used after a tunnel
// is established. Method names and encoder options are part of the stream
// format and must stay compatible with the original Crab clients.
package compress

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// Codec is a half-duplex compressor or decompressor wrapping a stream.
type Codec interface {
	io.ReadWriteCloser
}

type ReaderFunc func(r io.ReadCloser) (Codec, error)
type WriterFunc func(w io.WriteCloser) (Codec, error)

var (
	mu      sync.RWMutex
	readers = map[string]ReaderFunc{}
	writers = map[string]WriterFunc{}
)

func register(name string, rf ReaderFunc, wf WriterFunc) {
	name = strings.ToLower(name)
	mu.Lock()
	defer mu.Unlock()
	readers[name] = rf
	writers[name] = wf
}

func Lookup(name string) (ReaderFunc, WriterFunc, error) {
	name = strings.ToLower(name)
	mu.RLock()
	defer mu.RUnlock()
	rf, rok := readers[name]
	wf, wok := writers[name]
	if !rok || !wok {
		return nil, nil, fmt.Errorf("compress method %s not found", name)
	}
	return rf, wf, nil
}
