package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"fmt"
	"io"
)

type Aes128CfbCrypto struct {
	r io.Reader
	w io.WriteCloser
	c io.Closer

	ivRecv bool
	block  cipher.Block
}

func (c *Aes128CfbCrypto) Read(buf []byte) (int, error) {
	if !c.ivRecv {
		iv := make([]byte, aes.BlockSize)
		n, err := c.r.Read(iv)
		if err != nil {
			return n, err
		}
		if n != aes.BlockSize {
			return n, fmt.Errorf("read iv failed: want %d got %d", aes.BlockSize, n)
		}
		stream := cipher.NewCFBDecrypter(c.block, iv)
		c.r = cipher.StreamReader{S: stream, R: c.r}
		c.ivRecv = true
	}
	return c.r.Read(buf)
}

func (c *Aes128CfbCrypto) Write(buf []byte) (int, error) {
	return c.w.Write(buf)
}

func (c *Aes128CfbCrypto) Close() error {
	if c.c != nil {
		c.c.Close()
	}
	if c.w != nil {
		c.w.Close()
	}
	return nil
}

func NewAes128CfbCryptoReader(key string, r io.ReadCloser) (Crypto, error) {
	keyBuf := md5.Sum([]byte(key))

	block, err := aes.NewCipher(keyBuf[:])
	if err != nil {
		return nil, err
	}

	return &Aes128CfbCrypto{
		r:     r,
		w:     nil,
		block: block,
		c:     r,
	}, nil
}

func NewAes128CfbCryptoWriter(key string, w io.WriteCloser) (Crypto, error) {
	keyBuf := md5.Sum([]byte(key))
	iv, err := getRandomBytes(aes.BlockSize)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(keyBuf[:])
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)

	w.Write(iv)

	writer := &cipher.StreamWriter{S: stream, W: w}

	return &Aes128CfbCrypto{
		r: nil,
		w: writer,
		c: w,
	}, nil
}

func init() {
	registerCrypto("aes-128-cfb", NewAes128CfbCryptoReader, NewAes128CfbCryptoWriter)
}
