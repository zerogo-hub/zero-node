package network

import (
	"time"

	zerocompress "github.com/zerogo-hub/zero-helper/compress"
	zerologger "github.com/zerogo-hub/zero-helper/logger"
)

// Config 一些参数配置
type Config struct {
	// --------------------------- 服务 ---------------------------

	// MaxConnNum 连接数量上限，超过数量则拒绝连接
	// 负数表示不限制
	MaxConnNum int

	// Network 可选 "tcp", "tcp4", "tcp6"
	Network string
	Host    string
	Port    int

	Logger zerologger.Logger
	// loggerLevel 日志级别
	// 见 https://github.com/zerogo-hub/zero-helper/blob/main/logger/logger.go
	LoggerLevel int

	// OnServerStart 服务器启动时触发，套接字监听此时尚未启动
	OnServerStart func() error

	// OnServerClose 服务端关闭时触发，此时已关闭客户端连接
	OnServerClose func()

	// CloseTimeout 关闭服务器的等待时间，超过该时间服务器直接关闭
	CloseTimeout time.Duration

	// --------------------------- 会话 ---------------------------

	// RecvBufferSize 在 session 中接收消息 buffer 大小
	RecvBufferSize int

	// RecvDeadLine 通信超时时间，最终调用 conn.SetReadDeadline
	RecvDeadLine time.Duration

	// RecvQueueSize 在 session 中接收到的消息队列大小，session 接收到消息后并非立即处理，而是丢到一个消息队列中，异步处理
	RecvQueueSize int

	// SendBufferSize 发送消息 buffer 大小
	SendBufferSize int

	// SendDeadLine SendDeadline
	SendDeadLine time.Duration

	// SendQueueSize 发送的消息队列大小，消息优先发送到 sesion 的消息队列，然后写入到套接字中
	SendQueueSize int

	// OnConnected 客户端连接到来时触发，此时客户端已经可以开始收发消息
	OnConnected ConnFunc

	// OnConnClose 客户端连接关闭触发，此时客户端不可以再收发消息
	OnConnClose ConnFunc

	// Datapack 封包与解包
	Datapack Datapack

	// --------------------------- 封包与解包 ---------------------------

	// WhetherCompress 是否需要对消息负载进行压缩
	WhetherCompress bool

	// WhetherCrypto 是否需要对消息负载进行加密
	WhetherCrypto bool

	// CompressThreshold 压缩的阈值，当消息负载长度超过该值时才会压缩
	CompressThreshold int

	// Compress 压缩与解压器
	Compress zerocompress.Compress
}

// DefaultConfig 默认值
func DefaultConfig() *Config {
	config := &Config{
		MaxConnNum:     -1,
		Network:        "tcp4",
		Host:           "127.0.0.1",
		Port:           8001,
		Logger:         zerologger.NewSampleLogger(),
		LoggerLevel:    zerologger.DEBUG,
		RecvBufferSize: 8 * 1024,
		RecvQueueSize:  128,
		SendBufferSize: 8 * 1024,
		SendQueueSize:  128,
		CloseTimeout:   5 * time.Second,
	}

	return config
}

// Option 设置配置选项
type Option func(Peer)

// WithMaxConnNum 连接数量上限，超过数量则拒绝连接
// 负数表示不限制
func WithMaxConnNum(MaxConnNum int) Option {
	return func(p Peer) {
		p.SetMaxConnNum(MaxConnNum)
	}
}

// WithNetwork 可选 "tcp", "tcp4", "tcp6"
func WithNetwork(network string) Option {
	return func(p Peer) {
		p.SetNetwork(network)
	}
}

// WithHost 设置监听地址
func WithHost(host string) Option {
	return func(p Peer) {
		p.SetHost(host)
	}
}

// WithPort 设置监听端口
func WithPort(port int) Option {
	return func(p Peer) {
		p.SetPort(port)
	}
}

// WithLogger 设置日志
func WithLogger(logger zerologger.Logger) Option {
	return func(p Peer) {
		p.SetLogger(logger)
	}
}

// WithLoggerLevel 设置日志级别
// 见 https://github.com/zerogo-hub/zero-helper/blob/main/logger/logger.go
func WithLoggerLevel(loggerLevel int) Option {
	return func(p Peer) {
		p.SetLoggerLevel(loggerLevel)
		if p.Logger() != nil {
			p.Logger().SetLevel(loggerLevel)
		}
	}
}

// WithOnServerStart 服务器启动时触发，套接字监听此时尚未启动
func WithOnServerStart(onServerStart func() error) Option {
	return func(p Peer) {
		p.SetOnServerStart(onServerStart)
	}
}

// WithOnServerClose 服务端关闭时触发，此时已关闭客户端连接
func WithOnServerClose(onServerClose func()) Option {
	return func(p Peer) {
		p.SetOnServerClose(onServerClose)
	}
}

// WithCloseTimeout 关闭服务器的等待时间，超过该时间服务器直接关闭
func WithCloseTimeout(closeTimeout time.Duration) Option {
	return func(p Peer) {
		p.SetCloseTimeout(closeTimeout)
	}
}

// WithRecvBufferSize 在 session 中接收消息 buffer 大小
func WithRecvBufferSize(recvBufferSize int) Option {
	return func(p Peer) {
		p.SetRecvBufferSize(recvBufferSize)
	}
}

// WithRecvDeadLine 通信超时时间，最终调用 conn.SetReadDeadline
func WithRecvDeadLine(recvDeadLine time.Duration) Option {
	return func(p Peer) {
		p.SetRecvDeadLine(recvDeadLine)
	}
}

// WithRecvQueueSize 在 session 中接收到的消息队列大小，session 接收到消息后并非立即处理，而是丢到一个消息队列中，异步处理
func WithRecvQueueSize(recvQueueSize int) Option {
	return func(p Peer) {
		p.SetRecvBufferSize(recvQueueSize)
	}
}

// WithSendBufferSize 发送消息 buffer 大小
func WithSendBufferSize(sendBufferSize int) Option {
	return func(p Peer) {
		p.SetSendBufferSize(sendBufferSize)
	}
}

// WithSendDeadLine SendDeadline
func WithSendDeadLine(sendDeadLine time.Duration) Option {
	return func(p Peer) {
		p.SetSendDeadLine(sendDeadLine)
	}
}

// WithSendQueueSize 发送的消息队列大小，消息优先发送到 sesion 的消息队列，然后写入到套接字中
func WithSendQueueSize(sendQueueSize int) Option {
	return func(p Peer) {
		p.SetSendQueueSize(sendQueueSize)
	}
}

// WithOnConnected 客户端连接到来时触发，此时客户端已经可以开始收发消息
func WithOnConnected(onConnected ConnFunc) Option {
	return func(p Peer) {
		p.SetOnConnected(onConnected)
	}
}

// WithOnConnClose 客户端连接关闭触发，此时客户端不可以再收发消息
func WithOnConnClose(onConnClose ConnFunc) Option {
	return func(p Peer) {
		p.SetOnConnClose(onConnClose)
	}
}

// WithDatapack 封包与解包
func WithDatapack(datapack Datapack) Option {
	return func(p Peer) {
		p.SetDatapack(datapack)
	}
}

// WithWhetherCompress 是否需要对消息负载进行压缩
func WithWhetherCompress(whetherCompress bool) Option {
	return func(p Peer) {
		p.SetWhetherCompress(whetherCompress)
	}
}

// WithWhetherCrypto 是否需要对消息负载进行加密
func WithWhetherCrypto(whetherCrypto bool) Option {
	return func(p Peer) {
		p.SetWhetherCrypto(whetherCrypto)
	}
}

// WithCompressThreshold 压缩的阈值，当消息负载长度超过该值时才会压缩
func WithCompressThreshold(compressThreshold int) Option {
	return func(p Peer) {
		p.SetCompressThreshold(compressThreshold)
	}
}

// WithCompress 压缩与解压器
func WithCompress(compress zerocompress.Compress) Option {
	return func(p Peer) {
		p.SetCompress(compress)
	}
}
