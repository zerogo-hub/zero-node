package network

import (
	"net"
	"time"

	zerocircle "github.com/zerogo-hub/zero-helper/buffer/circle"
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

// NewMessageFunc 创建一条消息
type NewMessageFunc func(flag, sn, code uint16, module, action uint8, payload []byte) Message

// Peer 服务接口，比如表示 tcp 服务，udp 服务，websocket 服务
type Peer interface {
	// Start 开启服务
	Start() error

	// Close 关闭服务，释放资源
	Close() error

	// Logger 日志
	Logger() zerologger.Logger

	// Router 路由器
	Router() Router

	// SessionManager 会话管理器
	SessionManager() SessionManager

	PeerOption
}

// PeerOption peer 的一些配置表设置
type PeerOption interface {
	// WithOption 设置配置
	WithOption(opts ...Option) Peer

	// SetMaxConnNum 连接数量上限，超过数量则拒绝连接
	// 负数表示不限制
	SetMaxConnNum(MaxConnNum int)
	// SetNetwork 可选 "tcp", "tcp4", "tcp6"
	SetNetwork(network string)
	// SetHost 设置监听地址
	SetHost(host string)
	// SetPort 设置监听端口
	SetPort(port int)
	// SetLogger 设置日志
	SetLogger(logger zerologger.Logger)
	// SetLoggerLevel 设置日志级别
	// 见 https://github.com/zerogo-hub/zero-helper/blob/main/logger/logger.go
	SetLoggerLevel(loggerLevel int)

	// SetOnServerStart 服务器启动时触发，套接字监听此时尚未启动
	SetOnServerStart(onServerStart func() error)
	// SetOnServerClose 服务端关闭时触发，此时已关闭客户端连接
	SetOnServerClose(onServerClose func())
	// SetCloseTimeout 关闭服务器的等待时间，超过该时间服务器直接关闭
	SetCloseTimeout(closeTimeout time.Duration)

	// SetRecvBufferSize 在 session 中接收消息 buffer 大小
	SetRecvBufferSize(recvBufferSize int)
	// SetRecvDeadLine 通信超时时间，最终调用 conn.SetReadDeadline
	SetRecvDeadLine(recvDeadLine time.Duration)
	// SetRecvQueueSize 在 session 中接收到的消息队列大小，session 接收到消息后并非立即处理，而是丢到一个消息队列中，异步处理
	SetRecvQueueSize(recvQueueSize int)

	// SetSendBufferSize 发送消息 buffer 大小
	SetSendBufferSize(recvBufferSize int)
	// SetSendDeadLine SendDeadline
	SetSendDeadLine(recvDeadLine time.Duration)
	// SetSendQueueSize 发送的消息队列大小，消息优先发送到 sesion 的消息队列，然后写入到套接字中
	SetSendQueueSize(recvQueueSize int)

	// SetOnConnected 客户端连接到来时触发，此时客户端已经可以开始收发消息
	SetOnConnected(onConnected ConnFunc)
	// SetOnConnClose 客户端连接关闭触发，此时客户端不可以再收发消息
	SetOnConnClose(onConnClose ConnFunc)

	// SetDatapack 封包与解包
	SetDatapack(datapack Datapack)

	// SetWhetherCompress 是否需要对消息负载进行压缩
	SetWhetherCompress(whetherCompress bool)
	// SetCompressThreshold 压缩的阈值，当消息负载长度超过该值时才会压缩
	SetCompressThreshold(compressThreshold int)
	// SetCompress 压缩与解压器
	SetCompress(compress zerocompress.Compress)
	// SetWhetherCrypto 是否需要对消息负载进行加密
	SetWhetherCrypto(whetherCrypto bool)
}

// Session 表示与客户端的一条连接，也称为会话
type Session interface {
	// Run 让当前连接开始工作，比如收发消息，一般用于连接成功之后
	Run()

	// Close 停止接收客户端消息，也不再接收服务端消息。当已接收的服务端消息发送完毕后，断开连接
	Close()

	// Send 发送消息给客户端
	Send(message Message) error

	// SendCallback 发送消息给客户端，发送之后响应回调函数
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
	// timeout 超时时间，如果超时仍未发送完已接收的服务端消息，也强行关闭连接
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
	Unpack(buffer *zerocircle.Circle, crypto Crypto) ([]Message, error)
}

// HandlerFunc 路由消息处理函数
type HandlerFunc func(message Message) (Message, error)

// Router 消息处理路由器
type Router interface {
	// AddRouter 添加路由
	AddRouter(module, action uint8, handle HandlerFunc) error

	// Handler 路由处理
	Handler(message Message) (Message, error)
}
