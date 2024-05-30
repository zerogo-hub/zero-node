package network

import (
	"net"
	"time"

	zeroringbytes "github.com/zerogo-hub/zero-helper/buffer/ringbytes"
	zerocompress "github.com/zerogo-hub/zero-helper/compress"
	zerologger "github.com/zerogo-hub/zero-helper/logger"
)

// SessionID 定义 Session id 类型
type SessionID = uint64

// ConnFunc 与客户端连接相关的响应函数
type ConnFunc func(session Session)

// SendCallbackFunc 发送消息的回调函数
type SendCallbackFunc func(session Session)

// CloseCallbackFunc 关闭会话后的回调函数
type CloseCallbackFunc func(session Session)

// MessageHander 处理客户端消息
type MessageHander func(message Message) (Message, error)

// Peer 服务接口，表示一种服务，比如表示 tcp 服务，udp 服务，websocket 服务
type Peer interface {
	// Start 开启服务，不会阻塞
	Start() error

	// Close 关闭服务，释放资源
	Close() error

	// Logger 日志
	Logger() zerologger.Logger

	// Router 路由器
	Router() Router

	// SessionManager 会话管理器
	SessionManager() SessionManager

	// ListenSignal 监听信号
	ListenSignal()

	PeerOption
}

// PeerOption 服务的一些配置
type PeerOption interface {
	// WithOption 设置配置
	WithOption(opts ...Option) Peer

	// SetMaxConnNum 连接数量上限，超过数量则拒绝连接
	// 负数表示不限制
	SetMaxConnNum(MaxConnNum int)
	// SetNetwork 可选 "tcp", "tcp4", "tcp6"，仅在 tcp peer 下有效
	SetNetwork(network string)
	// SetHost 设置监听地址
	// 默认 127.0.0.1
	SetHost(host string)
	// SetPort 设置监听端口
	// 默认 8001
	SetPort(port int)
	// SetLogger 设置日志
	SetLogger(logger zerologger.Logger)
	// SetLoggerLevel 设置日志级别
	// 见 https://github.com/zerogo-hub/zero-helper/blob/main/logger/logger.go
	SetLoggerLevel(loggerLevel int)

	// SetOnServerStart 服务器启动时触发，此时尚未启动套接字监听
	SetOnServerStart(onServerStart func() error)
	// SetOnServerClose 服务端关闭时触发，此时已关闭套接字监听，关闭所有客户端连接
	SetOnServerClose(onServerClose func())
	// SetCloseTimeout 关闭服务器的等待时间，超过该时间服务器直接关闭
	// 默认 5 秒
	SetCloseTimeout(closeTimeout time.Duration)

	// SetRecvBufferSize 在 session 中接收消息 buffer 大小，默认 8K(8 * 1024)
	SetRecvBufferSize(recvBufferSize int)
	// SetRecvDeadline 通信超时时间，最终调用 conn.SetReadDeadline 进行设置
	SetRecvDeadline(recvDeadLine time.Duration)
	// SetRecvQueueSize 在 session 中接收到的消息队列大小，session 接收到消息后并非立即处理，而是丢到一个消息队列中，异步处理
	// 默认 128 个，超过此值后会阻塞消息
	SetRecvQueueSize(recvQueueSize int)

	// SetSendBufferSize 发送消息 buffer 大小，默认 8K(8 * 1024)
	SetSendBufferSize(recvBufferSize int)
	// SetSendDeadline SendDeadline
	SetSendDeadline(recvDeadLine time.Duration)
	// SetSendQueueSize 发送的消息队列大小，消息优先发送到 sesion 的消息队列，然后写入到套接字中
	// 默认 128 个，超过此值后会阻塞消息
	SetSendQueueSize(recvQueueSize int)

	// SetOnConnected 客户端连接到来时触发，此时客户端已经可以开始收发消息
	SetOnConnected(onConnected ConnFunc)
	// SetOnConnClose 客户端连接关闭触发，此时客户端不可以再收发消息
	SetOnConnClose(onConnClose ConnFunc)

	// SetDatapack 封包与解包
	SetDatapack(datapack Datapack)

	// SetWhetherCompress 是否需要对消息负载进行压缩
	SetWhetherCompress(whetherCompress bool)
	// SetCompressThreshold 压缩的阈值，当消息负载长度不小于该值时才会压缩
	SetCompressThreshold(compressThreshold int)
	// SetCompress 设置压缩与解压器
	SetCompress(compress zerocompress.Compress)
	// SetWhetherCrypto 是否需要对消息负载进行加解密
	SetWhetherCrypto(whetherCrypto bool)
	// SetWhetherChecksum 是否启用校验值功能，默认 false
	SetWhetherChecksum(whetherChecksum bool)
}

// Session 表示与客户端的一条连接，也称为会话
type Session interface {
	// Run 让当前连接开始工作，比如收发消息，用于连接成功之后
	Run()

	// Close 停止接收客户端消息，也不再接收服务端消息。当已接收的服务端消息发送完毕后，断开连接
	Close()

	// Send 发送消息给客户端
	Send(message Message) error

	// SendCallback 发送消息给客户端，发送成功之后响应回调函数
	SendCallback(message Message, callback SendCallbackFunc) error

	// ID 获取 sessionID，每一条连接都分配有一个唯一的 id
	ID() SessionID

	// RemoteAddr 客户端地址信息
	RemoteAddr() net.Addr

	// Conn 获取原始的连接
	Conn() net.Conn

	// SetCrypto 设置加密解密的工具
	SetCrypto(crypto Crypto)

	// Config 配置
	Config() *Config

	// Get 获取自定义参数
	Get(key string) interface{}

	// Set 设置自定义参数，存储于此次会话中
	Set(key string, value interface{})
}

// Client 客户端，一般用来编写测试用例
type Client interface {
	Session

	// Connect 连接服务
	// network: tcp,tcp4,tcp6,ws,wss
	Connect(network, host string, port int) error

	// Logger 日志
	Logger() zerologger.Logger
}

// SessionManager 会话管理器
type SessionManager interface {
	// GenSessionID 生成新的会话 ID
	GenSessionID() SessionID

	// Add 添加 Session
	Add(session Session)

	// Del 移除 Session
	Del(sessionID SessionID)

	// Get(sessionID SessionID) (Session, error)
	Get(sessionID SessionID) (Session, error)

	// Len 获取当前 Session 数量
	Len() int

	// Close 当前所有连接停止接收客户端消息，不再接收服务端消息，当已接收的服务端消息发送完毕后，断开连接
	Close()

	// Send 发送消息给客户端
	Send(sessionID SessionID, message Message) error

	// SendCallback  发送消息个客户端，发送之后进行回调
	SendCallback(sessionID SessionID, message Message, callback SendCallbackFunc) error

	// SendAll 给所有客户端发送消息
	SendAll(message Message)
}

// Message 通讯消息
type Message interface {
	// SessionID 会话 ID，每一个连接都有一个唯一的会话 ID
	SessionID() SessionID

	// SetSessionID 设置 sessionID
	SetSessionID(sessionID SessionID)

	// ModuleID 功能模块，用来表示一个功能大类，比如商店、副本
	ModuleID() uint8

	// ActionID 功能细分，用来表示一个功能里面的具体功能，比如进入副本，退出副本
	ActionID() uint8

	// Flag 标记
	Flag() uint16

	// SN 自增编号
	SN() uint16

	// Code 错误码
	Code() uint16

	// Payload 负载
	Payload() []byte

	// Checksum 校验值
	Checksum() [16]byte

	// String 打印消息
	String() string
}

// Crypto 加密与解密接口
type Crypto interface {
	// Encrypt 加密
	Encrypt(in []byte) ([]byte, error)

	// Decrypt 解密
	Decrypt(in []byte) ([]byte, error)
}

// Datapack 通讯数据封包与解包
type Datapack interface {
	// HeadLen 消息头长度
	HeadLen() int

	// Pack 封包
	Pack(message Message, crypto Crypto) ([]byte, error)

	// Unpack 解包
	Unpack(buffer *zeroringbytes.RingBytes, crypto Crypto) ([]Message, error)
}

// HandlerFunc 路由消息处理函数
type HandlerFunc func(message Message) (Message, error)

// Router 消息处理路由器
type Router interface {
	// AddRouter 添加路由
	AddRouter(module, action uint8, handle HandlerFunc) error

	// Handler 路由处理
	Handler(message Message) (Message, error)

	// SetHandlerFunc 设置自定义路由处理函数
	SetHandlerFunc(handler HandlerFunc)
}
