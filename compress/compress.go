package compress

import (
	"fmt"
	"io"
	"strings"
)

type Compress interface {
	Read(buf []byte) (int, error)
	Write(buf []byte) (int, error)
	Close() error
}

type NewCompressReaderFunc func(r io.ReadCloser) (Compress, error)
type NewCompressWriterFunc func(w io.WriteCloser) (Compress, error)

var compressReaderFuncMap map[string]NewCompressReaderFunc
var compressWriterFuncMap map[string]NewCompressWriterFunc

func registerCompress(name string, rf NewCompressReaderFunc, wf NewCompressWriterFunc) {
	name = strings.ToLower(name)
	if compressReaderFuncMap == nil {
		compressReaderFuncMap = make(map[string]NewCompressReaderFunc)
	}
	if compressWriterFuncMap == nil {
		compressWriterFuncMap = make(map[string]NewCompressWriterFunc)
	}
	compressReaderFuncMap[name] = rf
	compressWriterFuncMap[name] = wf
}

func GetCompress(name string) (NewCompressReaderFunc, NewCompressWriterFunc, error) {
	name = strings.ToLower(name)
	rf, rfOk := compressReaderFuncMap[name]
	wf, wfOk := compressWriterFuncMap[name]
	if !wfOk || !rfOk {
		return nil, nil, fmt.Errorf("compress method %s not found", name)
	}
	return rf, wf, nil
}
