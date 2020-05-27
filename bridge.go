package main

import (
	"io"
)

const TcpBridgeBufferSize = 4096

func tcpBridge(a io.ReadCloser, b io.WriteCloser) {
	defer func() {
		a.Close()
		b.Close()
	}()
	buf := make([]byte, TcpBridgeBufferSize)
	for {
		n, err := a.Read(buf)
		if err != nil {
			return
		}
		b.Write(buf[:n])
	}
}
