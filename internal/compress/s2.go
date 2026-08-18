package compress

import (
	"io"

	"github.com/klauspost/compress/s2"
)

type s2Codec struct {
	r *s2.Reader
	w *s2.Writer
	c io.Closer
}

func (s *s2Codec) Read(p []byte) (int, error) { return s.r.Read(p) }

func (s *s2Codec) Write(p []byte) (int, error) {
	n, err := s.w.Write(p)
	if err != nil {
		return n, err
	}
	// Flush after every write so interactive traffic is not buffered.
	return n, s.w.Flush()
}

func (s *s2Codec) Close() error {
	var first error
	if s.w != nil {
		first = s.w.Close()
	}
	if s.c != nil {
		if err := s.c.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func newS2Writer(w io.WriteCloser) (Codec, error) {
	// WriterBetterCompression is the original stream parameter and must stay.
	return &s2Codec{w: s2.NewWriter(w, s2.WriterBetterCompression()), c: w}, nil
}

func newS2Reader(r io.ReadCloser) (Codec, error) {
	return &s2Codec{r: s2.NewReader(r), c: r}, nil
}

func init() {
	register("s2", newS2Reader, newS2Writer)
}
