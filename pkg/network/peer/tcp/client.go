package tcp

import (
	"fmt"
	"net"
	"time"

	zerocompress "github.com/zerogo-hub/zero-helper/compress"
	zerologger "github.com/zerogo-hub/zero-helper/logger"
	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
)

// client 实现 Session 和 Client  接口
type client struct {
	session
}

// NewClient 创建一个 tcp 客户端，测试使用
func NewClient(handler zeronetwork.HandlerFunc, opts ...ClientOption) zeronetwork.Client {
	c := &client{}

	c.config = defaultConfig()

	for _, opt := range opts {
		opt(c)
	}

	session := &session{
		config:    c.config,
		sessionID: 0,
		sendQueue: make(chan *sendElement, c.config.sendQueueSize),
		recvQueue: make(chan zeronetwork.Message, c.config.recvQueueSize),
		closeCh:   make(chan bool),
		handler:   handler,
	}
	c.session = *session

	return c
}

// Connect 连接服务
func (c *client) Connect(network, host string, port int) error {

	address := fmt.Sprintf("%s:%d", host, port)
	addr, err := net.ResolveTCPAddr(network, address)
	if err != nil {
		c.config.logger.Error(err.Error())
		return err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		c.config.logger.Error(err.Error())
		return err
	}

	c.session.conn = conn

	return nil
}

// Logger 日志
func (c *client) Logger() zerologger.Logger {
	return c.config.logger
}

// ClientOption 设置配置选项
type ClientOption func(*client)

// WithClientLogger 设置日志
func WithClientLogger(logger zerologger.Logger) ClientOption {
	return func(c *client) {
		c.config.logger = logger
	}
}

// WithClientLoggerLevel 设置日志级别
// 见 https://github.com/zerogo-hub/zero-helper/blob/main/logger/logger.go
// WithLogger 设置日志
func WithClientLoggerLevel(loggerLevel int) ClientOption {
	return func(c *client) {
		c.config.loggerLevel = loggerLevel
		if c.config.logger != nil {
			c.config.logger.SetLevel(loggerLevel)
		}
	}
}

// WithClientRecvDeadLine 通信超时时间，最终调用 conn.SetReadDeadline
func WithClientRecvDeadLine(recvDeadLine time.Duration) ClientOption {
	return func(c *client) {
		c.config.recvDeadLine = recvDeadLine
	}
}

// WithClientRecvQueueSize 在 session 中接收到的消息队列大小，session 接收到消息后并非立即处理，而是丢到一个消息队列中，异步处理
func WithClientRecvQueueSize(recvQueueSize int) ClientOption {
	return func(c *client) {
		c.config.recvQueueSize = recvQueueSize
	}
}

// WithClientSendBufferSize 发送消息 buffer 大小
func WithClientSendBufferSize(sendBufferSize int) ClientOption {
	return func(c *client) {
		c.config.sendBufferSize = sendBufferSize
	}
}

// WithClientSendDeadLine SendDeadline
func WithClientSendDeadLine(sendDeadLine time.Duration) ClientOption {
	return func(c *client) {
		c.config.sendDeadLine = sendDeadLine
	}
}

// WithClientSendQueueSize 发送的消息队列大小，消息优先发送到 sesion 的消息队列，然后写入到套接字中
func WithClientSendQueueSize(sendQueueSize int) ClientOption {
	return func(c *client) {
		c.config.sendQueueSize = sendQueueSize
	}
}

// WithClientOnConnected 客户端连接到来时触发，此时客户端已经可以开始收发消息
func WithClientOnConnected(onConnected zeronetwork.ConnFunc) ClientOption {
	return func(c *client) {
		c.config.onConnected = onConnected
	}
}

// WithClientOnConnClose 客户端连接关闭触发，此时客户端不可以再收发消息
func WithClientOnConnClose(onConnClose zeronetwork.ConnFunc) ClientOption {
	return func(c *client) {
		c.config.onConnClose = onConnClose
	}
}

// WithClientDatapack 封包与解包
func WithClientDatapack(datapack zeronetwork.Datapack) ClientOption {
	return func(c *client) {
		c.config.datapack = datapack
	}
}

// WithClientWhetherCompress 是否需要对消息负载进行压缩
func WithClientWhetherCompress(whetherCompress bool) ClientOption {
	return func(c *client) {
		c.config.whetherCompress = whetherCompress
	}
}

// WithClientWhetherCrypto 是否需要对消息负载进行加密
func WithClientWhetherCrypto(whetherCrypto bool) ClientOption {
	return func(c *client) {
		c.config.whetherCrypto = whetherCrypto
	}
}

// WithClientCompressThreshold 压缩的阈值，当消息负载长度超过该值时才会压缩
func WithClientCompressThreshold(compressThreshold int) ClientOption {
	return func(c *client) {
		c.config.compressThreshold = compressThreshold
	}
}

// WithClientCompress 压缩与解压器
func WithClientCompress(compress zerocompress.Compress) ClientOption {
	return func(c *client) {
		c.config.compress = compress
	}
}
