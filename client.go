package main

import (
	"github.com/crabkun/crab/compress"
	"github.com/crabkun/crab/config"
	"github.com/crabkun/crab/crypto"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"time"
)

var clientConfig *config.ClientConfig

func ClientMain(cfg *config.ClientConfig) {
	clientConfig = cfg

	for i := range cfg.Ports {
		go clientListenAt(cfg.Ports[i])
	}

	select {}
}

func clientListenAt(cfg *config.PortConfig) {
	l := log.WithFields(log.Fields{
		"port_mark": cfg.Mark,
		"listen_at": cfg.LocalAddress,
	})

	listener, err := net.Listen("tcp", cfg.LocalAddress)
	if err != nil {
		l.WithError(err).Fatalln("listen failed")
	}

	l.Infoln("port running...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			l.WithError(err).Errorln("accept new connection failed, retrying in 1s")
			time.Sleep(time.Second * 1)
			continue
		}

		go clientHandleNewConn(conn, cfg)
	}
}

func clientHandleNewConn(remote net.Conn, cfg *config.PortConfig) {
	l := log.WithFields(log.Fields{
		"port_mark":   cfg.Mark,
		"listen_at":   cfg.LocalAddress,
		"remote_addr": remote.RemoteAddr(),
	})

	l.WithFields(log.Fields{
		"remote_addr": remote.RemoteAddr(),
	}).Debugln("new connection coming")

	var err error
	ok := false
	defer func() {
		if !ok {
			remote.Close()
		}
	}()

	l = l.WithFields(log.Fields{
		"master": clientConfig.Master,
	})

	master, err := net.Dial("tcp", clientConfig.Master)
	if err != nil {
		l.WithError(err).Errorln("dial master failed")
		return
	}

	defer func() {
		if !ok {
			master.Close()
		}
	}()

	// handshake and match port key
	SendCommand(master, CommandClientHandshake, []byte(cfg.PortKey))

	for {
		buf, err := ReadFromSocket(master)
		if err != nil {
			if err != io.EOF {
				l.WithError(err).Errorln("read from master failed")
			} else {
				l.Errorln("master disconnected connection")
			}
			return
		}

		// get the resp code
		code := buf[0]
		buf = buf[1:]

		switch code {
		case ResponseCodeServerAcceptClient:
			er, ew, err := crypto.GetCrypto(cfg.EncryptMethod)
			if err != nil {
				// checked while validate configure
				panic(err)
			}
			cr, cw, err := compress.GetCompress(cfg.CompressMethod)
			if err != nil {
				panic(err)
			}

			remoteToMaster, err := ew(cfg.PortKey, master)
			if err != nil {
				l.WithError(err).Errorln("init encrypt failed")
				return
			}
			remoteToMaster, err = cw(remoteToMaster)
			if err != nil {
				l.WithError(err).Errorln("init compress failed")
				return
			}

			masterToRemote, err := er(cfg.PortKey, master)
			if err != nil {
				l.WithError(err).Errorln("init decrypt failed")
				return
			}
			masterToRemote, err = cr(masterToRemote)
			if err != nil {
				l.WithError(err).Errorln("init decompress failed")
				return
			}

			l.Debugln("client match server success")
			ok = true
			// tcp bridge
			go tcpBridge(remote, remoteToMaster)
			go tcpBridge(masterToRemote, remote)
			return
		case ResponseCodeServerRejectClient:
			if len(buf) == 0 {
				return
			}
			// reject reason
			switch buf[0] {
			case RejectCodePortKeyNotExist:
				l.WithFields(log.Fields{
					"port_key": cfg.PortKey,
				}).Errorln("server reported that the port key is not registered at server")
			case RejectCodePortKeyRemoteConnectFailed:
				l.Errorln("server reported that connect to local address failed")
			}
		case ResponseCodePortKeyNotExist:
			l.WithFields(log.Fields{
				"port_key": cfg.PortKey,
			}).Errorln("master reported that the port key is not registered at master")
		case ResponseCodePortKeyConnectTimeout:
			l.Errorln("master reported that connect to server timeout")
		default:
			l.WithFields(log.Fields{
				"code": int(code),
			}).Errorln("master response an unsupported code")
			return
		}
	}
}
