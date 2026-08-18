package compress

import "io"

type nullCodec struct {
	r io.ReadCloser
	w io.WriteCloser
}

func (n *nullCodec) Read(p []byte) (int, error)  { return n.r.Read(p) }
func (n *nullCodec) Write(p []byte) (int, error) { return n.w.Write(p) }

func (n *nullCodec) Close() error {
	var first error
	if n.w != nil {
		first = n.w.Close()
	}
	if n.r != nil {
		if err := n.r.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func newNullWriter(w io.WriteCloser) (Codec, error) { return &nullCodec{w: w}, nil }
func newNullReader(r io.ReadCloser) (Codec, error)  { return &nullCodec{r: r}, nil }

func init() {
	register("null", newNullReader, newNullWriter)
}
