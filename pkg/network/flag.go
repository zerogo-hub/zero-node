package network

// MessageHead 中的 Flag
const (
	// FlagCompress 负载 payload 被压缩
	FlagCompress = uint16(0x0001)

	// FlagEncrypt 负载 payload 被加密
	FlagEncrypt = uint16(0x0010)

	// FlagChecksum 消息开启校验值检查
	FlagChecksum = uint16(0x0100)

	// FlagZero 特殊协议处理，不会派发到上层自定义处理函数中
	FlagZero = uint16(0x1000)
)

const (
	// FlagZeroExchangeKeyRequest 秘钥交换请求
	FlagZeroExchangeKeyRequest = uint8(1)

	// FlagZeroExchangeKeyResponse 秘钥交换响应
	FlagZeroExchangeKeyResponse = uint8(2)

	// FlagZeroHeartBeat 心跳包
	FlagZeroHeartBeat = uint8(3)
)
