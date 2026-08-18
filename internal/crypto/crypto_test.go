package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"io"
	"testing"
)

func TestAES128CFBRoundTrip(t *testing.T) {
	t.Parallel()

	const key = "T6wjoGQaqfDFDs1tySHVe8RXYYxnjWo4"
	plain := []byte("hello from crab tunnel payload")

	var buf bytes.Buffer
	w, err := newAES128CFBWriter(key, nopWriteCloser{&buf})
	if err != nil {
		t.Fatalf("writer: %v", err)
	}
	if _, err := w.Write(plain); err != nil {
		t.Fatalf("write: %v", err)
	}

	raw := buf.Bytes()
	if len(raw) < aes.BlockSize {
		t.Fatalf("ciphertext too short: %d", len(raw))
	}

	// Manual decode with the published key schedule to lock the algorithm.
	sum := md5.Sum([]byte(key))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		t.Fatal(err)
	}
	iv := raw[:aes.BlockSize]
	//nolint:staticcheck // protocol compatibility
	stream := cipher.NewCFBDecrypter(block, iv)
	manual := make([]byte, len(raw)-aes.BlockSize)
	stream.XORKeyStream(manual, raw[aes.BlockSize:])
	if !bytes.Equal(manual, plain) {
		t.Fatalf("manual decrypt mismatch: %q", manual)
	}

	r, err := newAES128CFBReader(key, io.NopCloser(bytes.NewReader(raw)))
	if err != nil {
		t.Fatalf("reader: %v", err)
	}
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("got %q want %q", got, plain)
	}
}

func TestLookupUnknown(t *testing.T) {
	t.Parallel()
	if _, _, err := Lookup("chacha20"); err == nil {
		t.Fatal("expected error")
	}
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
