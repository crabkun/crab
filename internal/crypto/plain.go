package crypto

import "io"

type plain struct {
	r io.ReadCloser
	w io.WriteCloser
}

func (c *plain) Read(p []byte) (int, error) {
	return c.r.Read(p)
}

func (c *plain) Write(p []byte) (int, error) {
	return c.w.Write(p)
}

func (c *plain) Close() error {
	var first error
	if c.w != nil {
		first = c.w.Close()
	}
	if c.r != nil {
		if err := c.r.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func newPlainReader(_ string, r io.ReadCloser) (Cipher, error) {
	return &plain{r: r}, nil
}

func newPlainWriter(_ string, w io.WriteCloser) (Cipher, error) {
	return &plain{w: w}, nil
}

func init() {
	register("plain", newPlainReader, newPlainWriter)
}
