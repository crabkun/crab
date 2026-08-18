package netutil

import (
	"errors"
	"io"
	"net"
	"strings"
)

func errorsIsClosed(err error) bool {
	if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "use of closed network connection")
}
