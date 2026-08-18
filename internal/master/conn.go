package master

import (
	"net"
	"sync"

	"github.com/crabkun/crab/internal/protocol"
)

// lockedConn serializes control-plane writes. The original master wrote
// register responses and "new client coming" notices on the same socket
// from different goroutines.
type lockedConn struct {
	net.Conn
	mu sync.Mutex
}

func (c *lockedConn) WriteResponse(code byte, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return protocol.WriteResponse(c.Conn, code, data)
}

func (c *lockedConn) WriteCommand(command byte, data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return protocol.WriteCommand(c.Conn, command, data)
}
