package compress

import (
	"bytes"
	"io"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	methods := []string{"null", "s2", "zstd"}
	payload := bytes.Repeat([]byte("crab-compress-payload-"), 64)

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			rf, wf, err := Lookup(method)
			if err != nil {
				t.Fatal(err)
			}

			var buf bytes.Buffer
			w, err := wf(nopWriteCloser{&buf})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := w.Write(payload); err != nil {
				t.Fatal(err)
			}
			if err := w.Close(); err != nil {
				t.Fatal(err)
			}

			r, err := rf(io.NopCloser(bytes.NewReader(buf.Bytes())))
			if err != nil {
				t.Fatal(err)
			}
			got, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, payload) {
				t.Fatalf("round-trip mismatch: got %d bytes", len(got))
			}
		})
	}
}

type nopWriteCloser struct{ io.Writer }

func (nopWriteCloser) Close() error { return nil }
