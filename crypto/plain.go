package crypto

import "io"

type PlainCrypto struct {
	r io.ReadCloser
	w io.WriteCloser
}

func (c *PlainCrypto) Read(buf []byte) (int, error) {
	return c.r.Read(buf)
}

func (c *PlainCrypto) Write(buf []byte) (int, error) {
	return c.w.Write(buf)
}

func (c *PlainCrypto) Close() error {
	if c.w != nil {
		c.w.Close()
	}
	if c.r != nil {
		c.r.Close()
	}
	return nil
}

func NewPlainCryptoReader(key string, r io.ReadCloser) (Crypto, error) {
	return &PlainCrypto{r: r}, nil
}

func NewPlainCryptoWriter(key string, w io.WriteCloser) (Crypto, error) {
	return &PlainCrypto{w: w}, nil
}

func init() {
	registerCrypto("plain", NewPlainCryptoReader, NewPlainCryptoWriter)
}
