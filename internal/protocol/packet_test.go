package protocol

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"testing"
)

func TestWriteReadPacketRoundTrip(t *testing.T) {
	t.Parallel()

	payloads := [][]byte{
		{CommandServerHandshake, 'k', 'e', 'y'},
		{ResponseReady},
		{ResponsePortKeyRegSuccess, 'a', 'b', 'c'},
		bytes.Repeat([]byte{0x7f}, 1024),
	}

	for _, payload := range payloads {
		var buf bytes.Buffer
		if err := WritePacket(&buf, payload); err != nil {
			t.Fatalf("WritePacket: %v", err)
		}
		got, err := ReadPacket(&buf)
		if err != nil {
			t.Fatalf("ReadPacket: %v", err)
		}
		if !bytes.Equal(got, payload) {
			t.Fatalf("payload mismatch:\n got %x\nwant %x", got, payload)
		}
	}
}

func TestServerHandshakeWireBytes(t *testing.T) {
	t.Parallel()

	// 00 0b 01 "crabserver"
	want, err := hex.DecodeString("000b0163726162736572766572")
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := WriteCommand(&buf, CommandServerHandshake, []byte("crabserver")); err != nil {
		t.Fatalf("WriteCommand: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("handshake frame mismatch:\n got %x\nwant %x", buf.Bytes(), want)
	}
}

func TestNewClientPayload(t *testing.T) {
	t.Parallel()

	portKey := "T6wjoGQaqfDFDs1tySHVe8RXYYxnjWo4"
	guid := "550e8400-e29b-41d4-a716-446655440000"

	encoded := EncodeNewClient(portKey, guid)
	if encoded[0] != byte(len(portKey)) {
		t.Fatalf("len prefix = %d, want %d", encoded[0], len(portKey))
	}

	gotKey, gotGUID, err := DecodeNewClient(encoded)
	if err != nil {
		t.Fatalf("DecodeNewClient: %v", err)
	}
	if gotKey != portKey || gotGUID != guid {
		t.Fatalf("decoded (%q, %q), want (%q, %q)", gotKey, gotGUID, portKey, guid)
	}
}

func TestReadPacketRejectsZeroLength(t *testing.T) {
	t.Parallel()

	_, err := ReadPacket(bytes.NewReader([]byte{0x00, 0x00}))
	if !errors.Is(err, ErrEmptyPacket) {
		t.Fatalf("err = %v, want %v", err, ErrEmptyPacket)
	}
}

func TestReadPacketHandlesSplitReads(t *testing.T) {
	t.Parallel()

	var full bytes.Buffer
	if err := WriteResponse(&full, ResponseServerAcceptClient, nil); err != nil {
		t.Fatal(err)
	}

	r := &oneByteReader{r: bytes.NewReader(full.Bytes())}
	got, err := ReadPacket(r)
	if err != nil {
		t.Fatalf("ReadPacket: %v", err)
	}
	if got[0] != ResponseServerAcceptClient {
		t.Fatalf("code = %d, want %d", got[0], ResponseServerAcceptClient)
	}
}

func TestWritePacketTooLong(t *testing.T) {
	t.Parallel()

	err := WritePacket(io.Discard, make([]byte, MaxPacketSize+1))
	if !errors.Is(err, ErrPacketTooBig) {
		t.Fatalf("err = %v, want %v", err, ErrPacketTooBig)
	}
}

type oneByteReader struct {
	r io.Reader
}

func (r *oneByteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return r.r.Read(p[:1])
}
