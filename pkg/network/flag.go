package network

// MessageHead 中的 Flag
const (
	// FlagCompress 负载 payload 被压缩
	FlagCompress = uint16(0x0001)

	// FlagEncrypt 负载 payload 被加密
	FlagEncrypt = uint16(0x0010)

	// FlagTick 这是心跳包消息
	FlagTick = uint16(0x0100)

	// FlagInit 这是初始化消息
	// 用于连接时与客户端做一些初始化信息，例如通过 dh 协议交换密钥，用于后续的 rc4 加密
	FlagInit = uint16(0x1000)
)
