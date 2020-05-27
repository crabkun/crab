package main

import (
	"bytes"
	"encoding/binary"
	"github.com/crabkun/crab/config"
	"github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"sync"
	"time"
)

var portKeyMap map[string]net.Conn
var portKeyMapLock sync.RWMutex

var clientMap map[string]*client
var clientMapLock sync.Mutex

type client struct {
	Conn net.Conn
	C    chan int
}

var masterConfig *config.MasterConfig

func MasterMain(cfg *config.MasterConfig) {
	portKeyMap = make(map[string]net.Conn)
	clientMap = make(map[string]*client)

	masterConfig = cfg

	l := log.WithFields(log.Fields{
		"listen_at": cfg.ListenAt,
	})

	listener, err := net.Listen("tcp", cfg.ListenAt)
	if err != nil {
		l.WithError(err).Fatalln("listen failed")
	}

	l.Infoln("master running...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			l.WithError(err).Errorln("accept new connection failed, retrying in 1s")
			time.Sleep(time.Second)
			continue
		}

		// disconnect if not handshake in 3s
		conn.SetDeadline(time.Now().Add(time.Second * 3))

		go masterHandleNewConn(conn)
	}
}

func masterHandleNewConn(conn net.Conn) {
	ok := false
	defer func() {
		if !ok {
			conn.Close()
		}
	}()

	l := log.WithFields(log.Fields{
		"remote_addr": conn.RemoteAddr(),
	})

	l.Debugln("new connection coming")

	// waiting handshake
	buf, err := ReadFromSocket(conn)
	if err != nil {
		l.WithError(err).Debugln("read handshake failed")
		return
	}
	command := buf[0]
	buf = buf[1:]

	switch command {
	case CommandServerHandshake:
		if len(buf) == 0 {
			return
		}

		// cancel handshake deadline
		conn.SetDeadline(time.Time{})
		ok = true

		go masterHandleServer(conn, buf)
		return
	case CommandClientHandshake:
		if len(buf) == 0 {
			return
		}

		// cancel handshake deadline
		conn.SetDeadline(time.Time{})
		ok = true

		go masterConnectPortKey(conn, string(buf))
		return
	case CommandServerAcceptClientRequest:
		if len(buf) == 0 {
			return
		}

		// cancel handshake deadline
		conn.SetDeadline(time.Time{})
		ok = true

		go masterMatchClient(conn, string(buf))
	default:
		// unsupported command
		return
	}
}

func masterHandleServer(conn net.Conn, masterKeyBuf []byte) {
	l := log.WithFields(log.Fields{
		"remote_addr": conn.RemoteAddr(),
		"remote_type": "server",
	})

	if string(masterKeyBuf) != masterConfig.MasterKey {
		Response(conn, ResponseCodeMasterKeyMismatch, nil)
		conn.Close()
		return
	}
	if Response(conn, ResponseCodeReady, nil) != nil {
		conn.Close()
		return
	}
	defer func() {
		// unregister port key after server disconnected
		// todo optimize
		portKeyMapLock.Lock()
		for k, v := range portKeyMap {
			if v.RemoteAddr() == conn.RemoteAddr() {
				delete(portKeyMap, k)
			}
		}
		conn.Close()
		portKeyMapLock.Unlock()
	}()
	for {
		packet, err := ReadFromSocket(conn)
		if err != nil {
			if err != io.EOF {
				l.WithError(err).Debugln("read server packet failed")
			} else {
				l.Debugln("server disconnected")
			}
			return
		}

		command := packet[0]
		packet = packet[1:]

		switch command {
		case CommandServerRegisterPortKey:
			if len(packet) == 0 {
				return
			}
			code := masterRegisterPortKey(conn, string(packet))
			err = Response(conn, code, packet)
			if err != nil {
				return
			}
		case CommandServerRejectClientRequest:
			if len(packet) < 2 {
				return
			}
			// reject reason
			reason := packet[0]
			clientGuid := string(packet[1:])

			// get client from map
			clientMapLock.Lock()
			thisClient, ok := clientMap[clientGuid]
			if !ok {
				clientMapLock.Unlock()
				continue
			}

			// delete client and close the timeout waiting channel
			delete(clientMap, clientGuid)
			close(thisClient.C)

			clientMapLock.Unlock()

			Response(thisClient.Conn, ResponseCodeServerRejectClient, []byte{reason})
			thisClient.Conn.Close()
		default:
			// unsupported command
			return
		}
	}
}

func masterRegisterPortKey(c net.Conn, portKey string) uint8 {
	portKeyMapLock.Lock()
	defer portKeyMapLock.Unlock()

	_, exist := portKeyMap[portKey]
	if exist {
		return ResponseCodePortKeyExist
	}

	portKeyMap[portKey] = c
	log.WithFields(log.Fields{
		"remote_addr": c.RemoteAddr(),
		"port_key":    portKey,
	}).Debugln("new port key register success")
	return ResponseCodePortKeyRegSuccess
}

func masterConnectPortKey(conn net.Conn, portKey string) {
	l := log.WithFields(log.Fields{
		"remote_addr": conn.RemoteAddr(),
		"remote_type": "client",
	})

	// find port key
	portKeyMapLock.RLock()
	server, exist := portKeyMap[portKey]
	if !exist {
		portKeyMapLock.RUnlock()
		Response(conn, ResponseCodePortKeyNotExist, []byte(portKey))
		conn.Close()
		return
	}
	portKeyMapLock.RUnlock()

	// gen guid for client
	guid, err := uuid.NewV4()
	if err != nil {
		l.WithError(err).Errorln("gen uuid failed")
		conn.Close()
		return
	}
	guidStr := guid.String()

	clientMapLock.Lock()
	// todo delete this ?
	_, exist = clientMap[guidStr]
	if exist {
		clientMapLock.Unlock()
		conn.Close()
		return
	}

	thisClient := &client{
		Conn: conn,
		C:    make(chan int),
	}
	clientMap[guidStr] = thisClient
	clientMapLock.Unlock()

	ok := false
	defer func() {
		if !ok {
			conn.Close()

			clientMapLock.Lock()
			delete(clientMap, guidStr)
			clientMapLock.Unlock()
		}
	}()

	buf := bytes.NewBuffer(nil)
	binary.Write(buf, binary.BigEndian, uint8(len(portKey)))
	buf.WriteString(portKey)
	buf.WriteString(guidStr)

	// tell server to connect local addres and callback to master with client guid
	err = Response(server, ResponseCodeNewClientComing, buf.Bytes())
	if err != nil {
		return
	}

	// waiting server to connect local address
	select {
	case <-time.After(time.Second * 10):
		// oops, timeout
		Response(conn, ResponseCodePortKeyConnectTimeout, nil)
		return
	case <-thisClient.C:
		// ok = true is not meant server accept the client, maybe reject
		ok = true
		return
	}

}

func masterMatchClient(c net.Conn, clientGuid string) {
	ok := false
	defer func() {
		if !ok {
			c.Close()
		}
	}()

	clientMapLock.Lock()
	thisClient, exist := clientMap[clientGuid]
	if !exist {
		clientMapLock.Unlock()
		return
	}
	delete(clientMap, clientGuid)
	// close the timeout waiting channel
	close(thisClient.C)
	clientMapLock.Unlock()

	err := Response(thisClient.Conn, ResponseCodeServerAcceptClient, nil)
	if err != nil {
		return
	}

	ok = true

	// tcp bridge
	go tcpBridge(thisClient.Conn, c)
	go tcpBridge(c, thisClient.Conn)
}
