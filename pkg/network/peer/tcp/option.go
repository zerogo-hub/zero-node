package tcp

import (
	"time"

	zerocompress "github.com/zerogo-hub/zero-helper/compress"
	zerologger "github.com/zerogo-hub/zero-helper/logger"
	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
)

// Config 一些参数配置
type Config struct {
	// --------------------------- 服务 ---------------------------

	// maxConnNum 连接数量上限，超过数量则拒绝连接
	// 负数表示不限制
	maxConnNum int

	// network 可选 "tcp", "tcp4", "tcp6"
	network string
	host    string
	port    int

	logger zerologger.Logger
	// loggerLevel 日志级别
	// 见 https://github.com/zerogo-hub/zero-helper/blob/main/logger/logger.go
	loggerLevel int

	// onServerStart 服务器启动时触发，套接字监听此时尚未启动
	onServerStart func() error

	// onServerClose 服务端关闭时触发，此时已关闭客户端连接
	onServerClose func()

	// closeTimeout 关闭服务器的等待时间，超过该时间服务器直接关闭
	closeTimeout time.Duration

	// --------------------------- 会话 ---------------------------

	// recvBufferSize 在 session 中接收消息 buffer 大小
	recvBufferSize int

	// recvDeadLine 通信超时时间，最终调用 conn.SetReadDeadline
	recvDeadLine time.Duration

	// recvQueueSize 在 session 中接收到的消息队列大小，session 接收到消息后并非立即处理，而是丢到一个消息队列中，异步处理
	recvQueueSize int

	// sendBufferSize 发送消息 buffer 大小
	sendBufferSize int

	// sendDeadLine SendDeadline
	sendDeadLine time.Duration

	// sendQueueSize 发送的消息队列大小，消息优先发送到 sesion 的消息队列，然后写入到套接字中
	sendQueueSize int

	// onConnected 客户端连接到来时触发，此时客户端已经可以开始收发消息
	onConnected zeronetwork.ConnFunc

	// onConnClose 客户端连接关闭触发，此时客户端不可以再收发消息
	onConnClose zeronetwork.ConnFunc

	// datapack 封包与解包
	datapack zeronetwork.Datapack

	// --------------------------- 封包与解包 ---------------------------

	// whetherCompress 是否需要对消息负载进行压缩
	whetherCompress bool

	// whetherCrypto 是否需要对消息负载进行加密
	whetherCrypto bool

	// compressThreshold 压缩的阈值，当消息负载长度超过该值时才会压缩
	compressThreshold int

	// compress 压缩与解压器
	compress zerocompress.Compress
}

// defaultConfig 默认值
func defaultConfig() *Config {
	config := &Config{
		maxConnNum:     -1,
		network:        "tcp4",
		host:           "127.0.0.1",
		port:           8001,
		logger:         zerologger.NewSampleLogger(),
		loggerLevel:    zerologger.DEBUG,
		recvBufferSize: 8 * 1024,
		recvQueueSize:  128,
		sendBufferSize: 8 * 1024,
		sendQueueSize:  128,
		closeTimeout:   5 * time.Second,
	}

	config.datapack = newLTD(HeadLen(), config)

	return config
}

// Option 设置配置选项
type Option func(*server)

// WithMaxConnNum 连接数量上限，超过数量则拒绝连接
// 负数表示不限制
func WithMaxConnNum(maxConnNum int) Option {
	return func(s *server) {
		s.config.maxConnNum = maxConnNum
	}
}

// WithNetwork 可选 "tcp", "tcp4", "tcp6"
func WithNetwork(network string) Option {
	return func(s *server) {
		s.config.network = network
	}
}

// WithHost 设置监听地址
func WithHost(host string) Option {
	return func(s *server) {
		s.config.host = host
	}
}

// WithPort 设置监听端口
func WithPort(port int) Option {
	return func(s *server) {
		s.config.port = port
	}
}

// WithLogger 设置日志
func WithLogger(logger zerologger.Logger) Option {
	return func(s *server) {
		s.config.logger = logger
	}
}

// WithLoggerLevel 设置日志级别
// 见 https://github.com/zerogo-hub/zero-helper/blob/main/logger/logger.go
// WithLogger 设置日志
func WithLoggerLevel(loggerLevel int) Option {
	return func(s *server) {
		s.config.loggerLevel = loggerLevel
		if s.config.logger != nil {
			s.config.logger.SetLevel(loggerLevel)
		}
	}
}

// WithOnServerStart 服务器启动时触发，套接字监听此时尚未启动
func WithOnServerStart(onServerStart func() error) Option {
	return func(s *server) {
		s.config.onServerStart = onServerStart
	}
}

// WithOnServerClose 服务端关闭时触发，此时已关闭客户端连接
func WithOnServerClose(onServerClose func()) Option {
	return func(s *server) {
		s.config.onServerClose = onServerClose
	}
}

// WithCloseTimeout 关闭服务器的等待时间，超过该时间服务器直接关闭
func WithCloseTimeout(closeTimeout time.Duration) Option {
	return func(s *server) {
		s.config.closeTimeout = closeTimeout
	}
}

// WithRecvBufferSize 在 session 中接收消息 buffer 大小
func WithRecvBufferSize(recvBufferSize int) Option {
	return func(s *server) {
		s.config.recvBufferSize = recvBufferSize
	}
}

// WithRecvDeadLine 通信超时时间，最终调用 conn.SetReadDeadline
func WithRecvDeadLine(recvDeadLine time.Duration) Option {
	return func(s *server) {
		s.config.recvDeadLine = recvDeadLine
	}
}

// WithRecvQueueSize 在 session 中接收到的消息队列大小，session 接收到消息后并非立即处理，而是丢到一个消息队列中，异步处理
func WithRecvQueueSize(recvQueueSize int) Option {
	return func(s *server) {
		s.config.recvQueueSize = recvQueueSize
	}
}

// WithSendBufferSize 发送消息 buffer 大小
func WithSendBufferSize(sendBufferSize int) Option {
	return func(s *server) {
		s.config.sendBufferSize = sendBufferSize
	}
}

// WithSendDeadLine SendDeadline
func WithSendDeadLine(sendDeadLine time.Duration) Option {
	return func(s *server) {
		s.config.sendDeadLine = sendDeadLine
	}
}

// WithSendQueueSize 发送的消息队列大小，消息优先发送到 sesion 的消息队列，然后写入到套接字中
func WithSendQueueSize(sendQueueSize int) Option {
	return func(s *server) {
		s.config.sendQueueSize = sendQueueSize
	}
}

// WithOnConnected 客户端连接到来时触发，此时客户端已经可以开始收发消息
func WithOnConnected(onConnected zeronetwork.ConnFunc) Option {
	return func(s *server) {
		s.config.onConnected = onConnected
	}
}

// WithOnConnClose 客户端连接关闭触发，此时客户端不可以再收发消息
func WithOnConnClose(onConnClose zeronetwork.ConnFunc) Option {
	return func(s *server) {
		s.config.onConnClose = onConnClose
	}
}

// WithDatapack 封包与解包
func WithDatapack(datapack zeronetwork.Datapack) Option {
	return func(s *server) {
		s.config.datapack = datapack
	}
}

// WithWhetherCompress 是否需要对消息负载进行压缩
func WithWhetherCompress(whetherCompress bool) Option {
	return func(s *server) {
		s.config.whetherCompress = whetherCompress
	}
}

// WithWhetherCrypto 是否需要对消息负载进行加密
func WithWhetherCrypto(whetherCrypto bool) Option {
	return func(s *server) {
		s.config.whetherCrypto = whetherCrypto
	}
}

// WithCompressThreshold 压缩的阈值，当消息负载长度超过该值时才会压缩
func WithCompressThreshold(compressThreshold int) Option {
	return func(s *server) {
		s.config.compressThreshold = compressThreshold
	}
}

// WithCompress 压缩与解压器
func WithCompress(compress zerocompress.Compress) Option {
	return func(s *server) {
		s.config.compress = compress
	}
}
