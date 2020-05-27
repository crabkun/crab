package compress

import (
	"github.com/klauspost/compress/zstd"
	"io"
)

type Zstd struct {
	r *zstd.Decoder
	w *zstd.Encoder
	c io.Closer
}

func (z *Zstd) Read(buf []byte) (int, error) {
	return z.r.Read(buf)
}

func (z *Zstd) Write(buf []byte) (int, error) {
	n, err := z.w.Write(buf)
	if err != nil {
		return n, err
	}
	return n, z.w.Flush()
}

func (z *Zstd) Close() error {
	if z.w != nil {
		z.w.Close()
	}
	if z.c != nil {
		z.c.Close()
	}
	return nil
}

func NewZstdCompressWriter(w io.WriteCloser) (Compress, error) {
	zw, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil, err
	}
	return &Zstd{
		w: zw,
		c: w,
	}, nil
}

func NewZstdCompressReader(r io.ReadCloser) (Compress, error) {
	zr, err := zstd.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &Zstd{
		r: zr,
		c: r,
	}, nil
}

func init() {
	registerCompress("zstd", NewZstdCompressReader, NewZstdCompressWriter)
}
