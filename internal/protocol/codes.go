// Package protocol implements Crab's length-prefixed TCP control plane.
//
// The on-wire format is unchanged from the original implementation:
//
//	[2-byte big-endian length][payload]
//
// The first payload byte is either a command (client/server -> master) or a
// response code (master -> client/server).
package protocol

// Command bytes sent by server/client to master.
const (
	CommandServerHandshake           byte = 0x1
	CommandClientHandshake           byte = 0x2
	CommandServerRegisterPortKey     byte = 0x3
	CommandServerRejectClientRequest byte = 0x4
	CommandServerAcceptClientRequest byte = 0x5
)

// Response codes sent by master (or forwarded from server decisions).
const (
	ResponseReady                 byte = 0
	ResponseMasterKeyMismatch     byte = 1
	ResponsePortKeyExist          byte = 2
	ResponsePortKeyNotExist       byte = 3
	ResponsePortKeyRegSuccess     byte = 4
	ResponsePortKeyConnectTimeout byte = 5
	ResponseNewClientComing       byte = 6
	ResponseServerRejectClient    byte = 7
	ResponseServerAcceptClient    byte = 8
)

// Reject reasons carried after ResponseServerRejectClient / CommandServerRejectClientRequest.
const (
	RejectPortKeyNotExist            byte = 1
	RejectPortKeyRemoteConnectFailed byte = 2
)

// Timeouts match the original service so mixed old/new deployments behave the same.
const (
	HandshakeTimeout    = 3  // seconds
	AcceptWaitTimeout   = 10 // seconds
	LocalDialTimeout    = 8  // seconds
	ReconnectInterval   = 3  // seconds
	AcceptRetryInterval = 1  // seconds
	MaxPacketSize       = 0xFFFF
	PacketLengthSize    = 2
)
