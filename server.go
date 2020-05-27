package main

import (
	"github.com/crabkun/crab/compress"
	"github.com/crabkun/crab/config"
	"github.com/crabkun/crab/crypto"
	log "github.com/sirupsen/logrus"
	"net"
	"time"
)

var serverConfig *config.ServerConfig

func ServerMain(cfg *config.ServerConfig) {
	var err error
	serverConfig = cfg

	l := log.WithFields(log.Fields{
		"master": cfg.Master,
	})

ConnectMaster:
	c, err := net.Dial("tcp", cfg.Master)
	if err != nil {
		l.WithError(err).Errorln("dial master failed, reconnecting in 3s")
		time.Sleep(time.Second * 3)
		goto ConnectMaster
	}

	// disconnect if not handshake in 3s
	c.SetDeadline(time.Now().Add(time.Second * 3))

	SendCommand(c, CommandServerHandshake, []byte(cfg.MasterKey))
	for {
		buf, err := ReadFromSocket(c)
		if err != nil {
			l.WithError(err).Errorln("read from master failed, reconnecting in 3s")
			c.Close()
			time.Sleep(time.Second * 3)
			goto ConnectMaster
		}
		code := buf[0]
		buf = buf[1:]

		switch code {
		case ResponseCodeReady:
			c.SetDeadline(time.Time{})
			// handshake success, starting to register port key
			for _, v := range serverConfig.Ports {
				SendCommand(c, CommandServerRegisterPortKey, []byte(v.PortKey))
			}
		case ResponseCodeMasterKeyMismatch:
			l.Fatalln("master reported that master key mismatch")
		case ResponseCodePortKeyExist, ResponseCodePortKeyRegSuccess:
			portKey := string(buf)
			pk, ok := cfg.GetPort(portKey)
			lf := log.Fields{
				"port_key": portKey,
			}
			if ok {
				lf["port_mark"] = pk.Mark
			}
			if code == ResponseCodePortKeyExist {
				l.WithFields(lf).Errorln("failed to register port key because already registered")
			} else {
				l.WithFields(lf).Infoln("port key register success")
			}
		case ResponseCodeNewClientComing:
			if len(buf) < 4 {
				continue
			}

			portKeyLen := buf[0]
			buf = buf[1:]

			if portKeyLen == 0 || int(portKeyLen) >= len(buf) {
				continue
			}
			portKey := string(buf[:portKeyLen])
			clientGuid := string(buf[portKeyLen:])

			portCfg, exist := cfg.GetPort(portKey)
			if !exist {
				l.WithFields(log.Fields{
					"port_key":    portKey,
					"client_guid": clientGuid,
				}).Errorln("master return a non-existent port key")
				SendCommand(c, 0x4, append([]byte{RejectCodePortKeyNotExist}, clientGuid...))
				continue
			}

			l.WithFields(log.Fields{
				"port_key":    portKey,
				"client_guid": clientGuid,
				"port_mark":   portCfg.Mark,
			}).Debugln("new client come from master")

			go serverHandleNewClient(c, portCfg, clientGuid)
		default:
			l.WithFields(log.Fields{
				"code": int(code),
			}).Errorln("master response an unsupported code")
		}
	}
}

func serverHandleNewClient(master net.Conn, portCfg *config.PortConfig, guid string) {
	l := log.WithFields(log.Fields{
		"local_addr":  portCfg.LocalAddress,
		"master":      serverConfig.Master,
		"client_guid": guid,
		"port_mark":   portCfg.Mark,
	})

	remoteConn, err := net.DialTimeout("tcp", portCfg.LocalAddress, time.Second*8)
	if err != nil {
		l.WithError(err).Errorln("connect to local address failed")
		SendCommand(master, CommandServerRejectClientRequest, append([]byte{RejectCodePortKeyRemoteConnectFailed}, []byte(guid)...))
		return
	}
	newMasterConn, err := net.Dial("tcp", serverConfig.Master)
	if err != nil {
		l.WithError(err).Errorln("callback master failed")
		return
	}
	SendCommand(newMasterConn, CommandServerAcceptClientRequest, []byte(guid))

	l.Debugln("client match success")

	er, ew, err := crypto.GetCrypto(portCfg.EncryptMethod)
	if err != nil {
		// because check while validate configure
		panic(err)
	}
	cr, cw, err := compress.GetCompress(portCfg.CompressMethod)
	if err != nil {
		panic(err)
	}

	remoteToMaster, err := ew(portCfg.PortKey, newMasterConn)
	if err != nil {
		l.WithError(err).Errorln("init encrypt failed")
		return
	}
	remoteToMaster, err = cw(remoteToMaster)
	if err != nil {
		l.WithError(err).Errorln("init compress failed")
		return
	}

	masterToRemote, err := er(portCfg.PortKey, newMasterConn)
	if err != nil {
		l.WithError(err).Errorln("init decrypt failed")
		return
	}
	masterToRemote, err = cr(masterToRemote)
	if err != nil {
		l.WithError(err).Errorln("init decompress failed")
		return
	}

	// tcp bridge
	go tcpBridge(remoteConn, remoteToMaster)
	go tcpBridge(masterToRemote, remoteConn)
}
