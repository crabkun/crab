package compress

import (
	"github.com/klauspost/compress/s2"
	"io"
)

type S2 struct {
	r *s2.Reader
	w *s2.Writer
	c io.Closer
}

func (s *S2) Read(buf []byte) (int, error) {
	return s.r.Read(buf)
}

func (s *S2) Write(buf []byte) (int, error) {
	n, err := s.w.Write(buf)
	if err != nil {
		return n, err
	}
	return n, s.w.Flush()
}

func (s *S2) Close() error {
	if s.w != nil {
		s.w.Close()
	}
	if s.c != nil {
		s.c.Close()
	}
	return nil
}

func NewS2CompressWriter(w io.WriteCloser) (Compress, error) {
	sw := s2.NewWriter(w, s2.WriterBetterCompression())
	return &S2{
		w: sw,
		c: w,
	}, nil
}

func NewS2CompressReader(r io.ReadCloser) (Compress, error) {
	sr := s2.NewReader(r)
	return &S2{
		r: sr,
		c: r,
	}, nil
}

func init() {
	registerCompress("s2", NewS2CompressReader, NewS2CompressWriter)
}
