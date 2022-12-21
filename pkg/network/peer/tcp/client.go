package tcp

import (
	"fmt"
	"net"
	"time"

	zerocompress "github.com/zerogo-hub/zero-helper/compress"
	zerologger "github.com/zerogo-hub/zero-helper/logger"
	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
	zerodatapack "github.com/zerogo-hub/zero-node/pkg/network/datapack"
)

// client 实现 Session 和 Client  接口
// 定义见 pkg/network/network.go
type client struct {
	ss *session
}

// NewClient 创建一个 tcp 客户端，测试使用
func NewClient(handler zeronetwork.HandlerFunc, opts ...ClientOption) zeronetwork.Client {
	session := newSession(
		0,
		nil,
		zeronetwork.DefaultConfig(),
		nil,
		handler,
	)

	c := &client{ss: session}

	for _, opt := range opts {
		opt(c)
	}

	if c.Config().Datapack == nil {
		WithClientDatapack(zerodatapack.DefaultDatapck(c.Config()))(c)
	}

	return c
}

// Connect 连接服务
func (c *client) Connect(network, host string, port int) error {

	address := fmt.Sprintf("%s:%d", host, port)
	addr, err := net.ResolveTCPAddr(network, address)
	if err != nil {
		c.Config().Logger.Error(err.Error())
		return err
	}

	conn, err := net.DialTCP("tcp", nil, addr)
	if err != nil {
		c.Config().Logger.Error(err.Error())
		return err
	}

	c.ss.conn = conn

	return nil
}

// Logger 日志
func (c *client) Logger() zerologger.Logger {
	return c.Config().Logger
}

// Run 让当前连接开始工作，比如收发消息，一般用于连接成功之后
func (c *client) Run() {
	c.ss.Run()
}

// Close 停止接收客户端消息，也不再接收服务端消息。当已接收的服务端消息发送完毕后，断开连接
func (c *client) Close() {
	c.ss.Close()
}

// Send 发送消息给客户端
func (c *client) Send(message zeronetwork.Message) error {
	return c.ss.Send(message)
}

// SendCallback 发送消息给客户端，发送之后响应回调函数
func (c *client) SendCallback(message zeronetwork.Message, callback zeronetwork.SendCallbackFunc) error {
	return c.ss.SendCallback(message, callback)
}

// ID 获取 sessionID，每一条连接都分配有一个唯一的 id
func (c *client) ID() zeronetwork.SessionID {
	return c.ss.ID()
}

// RemoteAddr 客户端地址信息
func (c *client) RemoteAddr() net.Addr {
	return c.ss.RemoteAddr()
}

// Conn 获取原始的连接
func (c *client) Conn() net.Conn {
	return c.ss.Conn()
}

// SetCrypto 设置加密解密的工具
func (c *client) SetCrypto(crypto zeronetwork.Crypto) {
	c.ss.SetCrypto(crypto)
}

// Config 配置
func (c *client) Config() *zeronetwork.Config {
	return c.ss.Config()
}

// Get 获取自定义参数
func (c *client) Get(key string) interface{} {
	return c.ss.Get(key)
}

// Set 设置自定义参数
func (c *client) Set(key string, value interface{}) {
	c.ss.Set(key, value)
}

// ClientOption 设置配置选项
type ClientOption func(*client)

// WithClientLogger 设置日志
func WithClientLogger(logger zerologger.Logger) ClientOption {
	return func(c *client) {
		c.Config().Logger = logger
	}
}

// WithClientLoggerLevel 设置日志级别
// 见 https://github.com/zerogo-hub/zero-helper/blob/main/logger/logger.go
// WithLogger 设置日志
func WithClientLoggerLevel(loggerLevel int) ClientOption {
	return func(c *client) {
		c.Config().LoggerLevel = loggerLevel
		if c.Config().Logger != nil {
			c.Config().Logger.SetLevel(loggerLevel)
		}
	}
}

// WithClientRecvDeadLine 通信超时时间，最终调用 conn.SetReadDeadline
func WithClientRecvDeadLine(recvDeadLine time.Duration) ClientOption {
	return func(c *client) {
		c.Config().RecvDeadLine = recvDeadLine
	}
}

// WithClientRecvQueueSize 在 session 中接收到的消息队列大小，session 接收到消息后并非立即处理，而是丢到一个消息队列中，异步处理
func WithClientRecvQueueSize(recvQueueSize int) ClientOption {
	return func(c *client) {
		c.Config().RecvQueueSize = recvQueueSize
	}
}

// WithClientSendBufferSize 发送消息 buffer 大小
func WithClientSendBufferSize(sendBufferSize int) ClientOption {
	return func(c *client) {
		c.Config().SendBufferSize = sendBufferSize
	}
}

// WithClientSendDeadLine SendDeadline
func WithClientSendDeadLine(sendDeadLine time.Duration) ClientOption {
	return func(c *client) {
		c.Config().SendDeadLine = sendDeadLine
	}
}

// WithClientSendQueueSize 发送的消息队列大小，消息优先发送到 sesion 的消息队列，然后写入到套接字中
func WithClientSendQueueSize(sendQueueSize int) ClientOption {
	return func(c *client) {
		c.Config().SendQueueSize = sendQueueSize
	}
}

// WithClientOnConnected 客户端连接到来时触发，此时客户端已经可以开始收发消息
func WithClientOnConnected(onConnected zeronetwork.ConnFunc) ClientOption {
	return func(c *client) {
		c.Config().OnConnected = onConnected
	}
}

// WithClientOnConnClose 客户端连接关闭触发，此时客户端不可以再收发消息
func WithClientOnConnClose(onConnClose zeronetwork.ConnFunc) ClientOption {
	return func(c *client) {
		c.Config().OnConnClose = onConnClose
	}
}

// WithClientDatapack 封包与解包
func WithClientDatapack(datapack zeronetwork.Datapack) ClientOption {
	return func(c *client) {
		c.Config().Datapack = datapack
	}
}

// WithClientWhetherCompress 是否需要对消息负载进行压缩
func WithClientWhetherCompress(whetherCompress bool) ClientOption {
	return func(c *client) {
		c.Config().WhetherCompress = whetherCompress
	}
}

// WithClientWhetherCrypto 是否需要对消息负载进行加密
func WithClientWhetherCrypto(whetherCrypto bool) ClientOption {
	return func(c *client) {
		c.Config().WhetherCrypto = whetherCrypto
	}
}

// WithClientCompressThreshold 压缩的阈值，当消息负载长度超过该值时才会压缩
func WithClientCompressThreshold(compressThreshold int) ClientOption {
	return func(c *client) {
		c.Config().CompressThreshold = compressThreshold
	}
}

// WithClientCompress 压缩与解压器
func WithClientCompress(compress zerocompress.Compress) ClientOption {
	return func(c *client) {
		c.Config().Compress = compress
	}
}
