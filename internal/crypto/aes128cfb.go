package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"io"
)

// aes-128-cfb with key = MD5(port_key) and a random 16-byte IV prepended by
// the writer. CFB+MD5 is retained solely for wire compatibility with the
// original protocol; it is not a modern construction.
type aes128cfb struct {
	r      io.Reader
	w      io.WriteCloser
	c      io.Closer
	block  cipher.Block
	ivRecv bool
}

func (c *aes128cfb) Read(p []byte) (int, error) {
	if !c.ivRecv {
		iv := make([]byte, aes.BlockSize)
		if _, err := io.ReadFull(c.r, iv); err != nil {
			return 0, fmt.Errorf("read iv failed: %w", err)
		}
		//nolint:staticcheck // CFB is the published Crab data-plane cipher.
		stream := cipher.NewCFBDecrypter(c.block, iv)
		c.r = cipher.StreamReader{S: stream, R: c.r}
		c.ivRecv = true
	}
	return c.r.Read(p)
}

func (c *aes128cfb) Write(p []byte) (int, error) {
	return c.w.Write(p)
}

func (c *aes128cfb) Close() error {
	var first error
	if c.w != nil {
		first = c.w.Close()
	}
	if c.c != nil {
		if err := c.c.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func newAES128CFBReader(key string, r io.ReadCloser) (Cipher, error) {
	sum := md5.Sum([]byte(key))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	return &aes128cfb{r: r, block: block, c: r}, nil
}

func newAES128CFBWriter(key string, w io.WriteCloser) (Cipher, error) {
	sum := md5.Sum([]byte(key))
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(iv); err != nil {
		return nil, fmt.Errorf("write iv failed: %w", err)
	}
	//nolint:staticcheck // CFB is the published Crab data-plane cipher.
	stream := cipher.NewCFBEncrypter(block, iv)
	return &aes128cfb{
		w: &cipher.StreamWriter{S: stream, W: w},
		c: w,
	}, nil
}

func init() {
	register("aes-128-cfb", newAES128CFBReader, newAES128CFBWriter)
}
