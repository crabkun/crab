package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

func ReadFromSocket(c net.Conn) ([]byte, error) {
	l := make([]byte, 2)

	n, err := c.Read(l)
	if err != nil {
		return nil, err
	}
	if n != 2 {
		return nil, fmt.Errorf("read packet length failed, want 2 got %d", n)
	}

	length := binary.BigEndian.Uint16(l)
	if length == 0 {
		return nil, fmt.Errorf("read packet length failed, got zero length")
	}

	buf := make([]byte, length)

	n, err = c.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read packet failed, want %d got %d", length, n)
	}
	if uint16(n) != length {
		return nil, fmt.Errorf("read packet length failed, want 2 got %d", n)
	}

	return buf, nil
}

func WriteToSocket(c net.Conn, data []byte) error {
	if len(data) > 0xFFFF {
		return fmt.Errorf("packet too long")
	}

	lenBuf := make([]byte, 2)
	binary.BigEndian.PutUint16(lenBuf, uint16(len(data)))

	buf := bytes.NewBuffer(lenBuf)
	buf.Write(data)

	n, err := c.Write(buf.Bytes())
	if err != nil {
		return err
	}
	if n != buf.Len() {
		return fmt.Errorf("write packet failed, packet length %d sent %d", buf.Len(), n)
	}

	return nil
}

func Response(c net.Conn, code uint8, data []byte) error {
	return WriteToSocket(c, append([]byte{code}, data...))
}

func SendCommand(c net.Conn, command uint8, data []byte) error {
	return WriteToSocket(c, append([]byte{command}, data...))
}
