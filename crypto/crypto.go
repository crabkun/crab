package crypto

import (
	"fmt"
	"io"
	"strings"
)

type Crypto interface {
	Read(buf []byte) (int, error)
	Write(buf []byte) (int, error)
	Close() error
}

type NewCryptoReaderFunc func(key string, r io.ReadCloser) (Crypto, error)
type NewCryptoWriterFunc func(key string, w io.WriteCloser) (Crypto, error)

var cryptoReaderFuncMap map[string]NewCryptoReaderFunc
var cryptoWriterFuncMap map[string]NewCryptoWriterFunc

func registerCrypto(name string, rf NewCryptoReaderFunc, wf NewCryptoWriterFunc) {
	name = strings.ToLower(name)
	if cryptoReaderFuncMap == nil {
		cryptoReaderFuncMap = make(map[string]NewCryptoReaderFunc)
	}
	if cryptoWriterFuncMap == nil {
		cryptoWriterFuncMap = make(map[string]NewCryptoWriterFunc)
	}
	cryptoReaderFuncMap[name] = rf
	cryptoWriterFuncMap[name] = wf
}

func GetCrypto(name string) (NewCryptoReaderFunc, NewCryptoWriterFunc, error) {
	name = strings.ToLower(name)
	rf, rfOk := cryptoReaderFuncMap[name]
	wf, wfOk := cryptoWriterFuncMap[name]
	if !wfOk || !rfOk {
		return nil, nil, fmt.Errorf("encrypt method %s not found", name)
	}
	return rf, wf, nil
}
