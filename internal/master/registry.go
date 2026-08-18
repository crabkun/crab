package master

import (
	"net"
	"sync"
	"time"
)

type serverSession struct {
	conn     *lockedConn
	portKeys map[string]struct{}
}

type pendingClient struct {
	conn net.Conn
	done chan struct{}
}

type registry struct {
	mu      sync.RWMutex
	ports   map[string]*serverSession
	pending map[string]*pendingClient
}

func newRegistry() *registry {
	return &registry{
		ports:   make(map[string]*serverSession),
		pending: make(map[string]*pendingClient),
	}
}

func (r *registry) registerPort(sess *serverSession, portKey string) (exists bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.ports[portKey]; ok {
		return true
	}
	r.ports[portKey] = sess
	sess.portKeys[portKey] = struct{}{}
	return false
}

func (r *registry) lookupServer(portKey string) (*serverSession, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	sess, ok := r.ports[portKey]
	return sess, ok
}

func (r *registry) dropServer(sess *serverSession) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key := range sess.portKeys {
		if current, ok := r.ports[key]; ok && current == sess {
			delete(r.ports, key)
		}
	}
	clear(sess.portKeys)
}

func (r *registry) putPending(guid string, client *pendingClient) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.pending[guid]; ok {
		return false
	}
	r.pending[guid] = client
	return true
}

func (r *registry) takePending(guid string) (*pendingClient, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.pending[guid]
	if ok {
		delete(r.pending, guid)
	}
	return c, ok
}

func (r *registry) deletePending(guid string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.pending, guid)
}

func waitDone(done <-chan struct{}, timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-timer.C:
		return false
	case <-done:
		return true
	}
}
