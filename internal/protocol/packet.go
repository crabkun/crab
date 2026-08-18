package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

var (
	ErrEmptyPacket  = errors.New("read packet length failed, got zero length")
	ErrPacketTooBig = errors.New("packet too long")
)

// ReadPacket reads one length-prefixed control packet using io.ReadFull so a
// split TCP segment cannot truncate the frame. The wire layout is identical
// to the original 2-byte big-endian length + payload format.
func ReadPacket(r io.Reader) ([]byte, error) {
	var header [PacketLengthSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint16(header[:])
	if length == 0 {
		return nil, ErrEmptyPacket
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("read packet failed, want %d: %w", length, err)
	}
	return buf, nil
}

// WritePacket writes one length-prefixed control packet.
func WritePacket(w io.Writer, data []byte) error {
	if len(data) > MaxPacketSize {
		return ErrPacketTooBig
	}

	buf := make([]byte, PacketLengthSize+len(data))
	binary.BigEndian.PutUint16(buf[:PacketLengthSize], uint16(len(data)))
	copy(buf[PacketLengthSize:], data)

	if _, err := w.Write(buf); err != nil {
		return err
	}
	return nil
}

func WriteCommand(w io.Writer, command byte, data []byte) error {
	return WritePacket(w, prepend(command, data))
}

func WriteResponse(w io.Writer, code byte, data []byte) error {
	return WritePacket(w, prepend(code, data))
}

func prepend(b byte, data []byte) []byte {
	out := make([]byte, 1+len(data))
	out[0] = b
	copy(out[1:], data)
	return out
}

// EncodeNewClient builds the ResponseNewClientComing payload:
//
//	[1-byte portKeyLen][portKey][clientGUID]
func EncodeNewClient(portKey, guid string) []byte {
	buf := make([]byte, 1+len(portKey)+len(guid))
	buf[0] = byte(len(portKey))
	copy(buf[1:], portKey)
	copy(buf[1+len(portKey):], guid)
	return buf
}

// DecodeNewClient parses a ResponseNewClientComing payload.
func DecodeNewClient(buf []byte) (portKey, guid string, err error) {
	if len(buf) < 2 {
		return "", "", fmt.Errorf("new-client payload too short: %d", len(buf))
	}
	portKeyLen := int(buf[0])
	rest := buf[1:]
	if portKeyLen == 0 || portKeyLen >= len(rest) {
		return "", "", fmt.Errorf("invalid port key length %d (payload %d)", portKeyLen, len(rest))
	}
	return string(rest[:portKeyLen]), string(rest[portKeyLen:]), nil
}

// SplitCode extracts the leading command/response byte.
func SplitCode(packet []byte) (byte, []byte, error) {
	if len(packet) == 0 {
		return 0, nil, errors.New("empty packet")
	}
	return packet[0], packet[1:], nil
}

// ClearDeadline removes a handshake deadline from a TCP connection.
func ClearDeadline(c net.Conn) {
	_ = c.SetDeadline(time.Time{})
}
