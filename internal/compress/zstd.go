package compress

import (
	"io"

	"github.com/klauspost/compress/zstd"
)

type zstdCodec struct {
	r *zstd.Decoder
	w *zstd.Encoder
	c io.Closer
}

func (z *zstdCodec) Read(p []byte) (int, error) { return z.r.Read(p) }

func (z *zstdCodec) Write(p []byte) (int, error) {
	n, err := z.w.Write(p)
	if err != nil {
		return n, err
	}
	return n, z.w.Flush()
}

func (z *zstdCodec) Close() error {
	var first error
	if z.w != nil {
		first = z.w.Close()
	}
	if z.r != nil {
		// Decoder.Close must only run after the Read side has returned.
		z.r.Close()
	}
	if z.c != nil {
		if err := z.c.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func newZstdWriter(w io.WriteCloser) (Codec, error) {
	// SpeedBestCompression is the original encoder level and must stay.
	enc, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
	if err != nil {
		return nil, err
	}
	return &zstdCodec{w: enc, c: w}, nil
}

func newZstdReader(r io.ReadCloser) (Codec, error) {
	dec, err := zstd.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &zstdCodec{r: dec, c: r}, nil
}

func init() {
	register("zstd", newZstdReader, newZstdWriter)
}
