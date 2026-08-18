// Package crypto implements the data-plane ciphers used after a tunnel is
// established. Algorithm names and key derivation are part of the wire
// protocol and must stay compatible with the original Crab clients.
package crypto

import (
	"fmt"
	"io"
	"strings"
	"sync"
)

// Cipher is a half-duplex encryptor or decryptor wrapping a stream.
type Cipher interface {
	io.ReadWriteCloser
}

type ReaderFunc func(key string, r io.ReadCloser) (Cipher, error)
type WriterFunc func(key string, w io.WriteCloser) (Cipher, error)

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

// Lookup returns the reader/writer constructors for a configured method.
func Lookup(name string) (ReaderFunc, WriterFunc, error) {
	name = strings.ToLower(name)
	mu.RLock()
	defer mu.RUnlock()
	rf, rok := readers[name]
	wf, wok := writers[name]
	if !rok || !wok {
		return nil, nil, fmt.Errorf("encrypt method %s not found", name)
	}
	return rf, wf, nil
}
