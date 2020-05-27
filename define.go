package main

const (
	ResponseCodeReady                 = 0
	ResponseCodeMasterKeyMismatch     = 1
	ResponseCodePortKeyExist          = 2
	ResponseCodePortKeyNotExist       = 3
	ResponseCodePortKeyRegSuccess     = 4
	ResponseCodePortKeyConnectTimeout = 5
	ResponseCodeNewClientComing       = 6
	ResponseCodeServerRejectClient    = 7
	ResponseCodeServerAcceptClient    = 8
)

const (
	RejectCodePortKeyNotExist            = 1
	RejectCodePortKeyRemoteConnectFailed = 2
)

const (
	CommandServerHandshake           = 0x1
	CommandClientHandshake           = 0x2
	CommandServerRegisterPortKey     = 0x3
	CommandServerRejectClientRequest = 0x4
	CommandServerAcceptClientRequest = 0x5
)
