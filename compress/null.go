package compress

import (
	"io"
)

type NullCompress struct {
	r io.ReadCloser
	w io.WriteCloser
}

func (n *NullCompress) Read(buf []byte) (int, error) {
	return n.r.Read(buf)
}

func (n *NullCompress) Write(buf []byte) (int, error) {
	return n.w.Write(buf)
}

func (n *NullCompress) Close() error {
	if n.r != nil {
		n.r.Close()
	}
	if n.w != nil {
		n.w.Close()
	}
	return nil
}

func NewNullCompressWriter(w io.WriteCloser) (Compress, error) {
	return &NullCompress{
		w: w,
	}, nil
}

func NewNullCompressReader(r io.ReadCloser) (Compress, error) {
	return &NullCompress{
		r: r,
	}, nil
}

func init() {
	registerCompress("null", NewNullCompressReader, NewNullCompressWriter)
}
