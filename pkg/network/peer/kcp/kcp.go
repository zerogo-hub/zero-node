package kcp

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	kcp "github.com/xtaci/kcp-go/v5"

	zerocompress "github.com/zerogo-hub/zero-helper/compress"
	zerologger "github.com/zerogo-hub/zero-helper/logger"
	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
	zerodatapack "github.com/zerogo-hub/zero-node/pkg/network/datapack"
)

// server kcp 服务
// 实现接口: Peer
type server struct {
	// config 通用配置
	config *zeronetwork.Config

	// kcpConfig KCP 专属配置
	kcpConfig *Config

	// ln 监听套接字
	ln *kcp.Listener

	// sessionManager 会话管理
	sessionManager zeronetwork.SessionManager

	// closeOnce 防止多次关闭服务
	closeOnce sync.Once

	// isClosed 服务器已关闭
	isClosed bool

	// isCloseConn 服务器不再接收新连接
	isCloseConn bool

	// router 路由
	router zeronetwork.Router
}

// NewServer 创建一个 tcp 服务
func NewServer(opts ...Option) zeronetwork.Peer {
	s := &server{
		config:         zeronetwork.DefaultConfig(),
		kcpConfig:      defaultConfig(),
		sessionManager: zeronetwork.NewSessionManager(),
		router:         zeronetwork.NewRouter(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// WithOption 设置配置
func (s *server) WithOption(opts ...zeronetwork.Option) zeronetwork.Peer {
	for _, opt := range opts {
		opt(s)
	}

	if s.config.Datapack == nil {
		s.config.Datapack = zerodatapack.DefaultDatapck(s.config)
	}

	return s
}

// Start 开启服务
func (s *server) Start() error {
	if s.config.OnServerStart != nil {
		if err := s.config.OnServerStart(); err != nil {
			return err
		}
	}

	go s.listen()

	return nil
}

// Close 关闭服务，释放资源
func (s *server) Close() error {
	var once bool

	s.closeOnce.Do(func() {
		once = true
	})

	if once {
		ctx, cancel := context.WithTimeout(context.Background(), s.config.CloseTimeout)
		defer cancel()

		ch := make(chan bool)

		go func() {
			s.isClosed = true
			s.isCloseConn = true

			// 停止监听
			if err := s.ln.Close(); err != nil {
				s.config.Logger.Errorf("close listen failed: %s", err.Error())
			}

			// 关闭所有连接
			s.sessionManager.Close()

			// 处理自定义行为
			if s.config.OnServerClose != nil {
				s.config.OnServerClose()
			}

			ch <- true
		}()

		select {
		case <-ch:
			s.config.Logger.Info("close success")
			break
		case <-ctx.Done():
			s.config.Logger.Error("close timeout")
			break
		}
	}

	return nil
}

// Logger 日志
func (s *server) Logger() zerologger.Logger {
	return s.config.Logger
}

// Router 路由器
func (s *server) Router() zeronetwork.Router {
	return s.router
}

// SessionManager 会话管理器
func (s *server) SessionManager() zeronetwork.SessionManager {
	return s.sessionManager
}

// SetMaxConnNum 连接数量上限，超过数量则拒绝连接
// 负数表示不限制
func (s *server) SetMaxConnNum(MaxConnNum int) {
	s.config.MaxConnNum = MaxConnNum
}

// SetNetwork 可选 "tcp", "tcp4", "tcp6"
func (s *server) SetNetwork(network string) {

}

// SetHost 设置监听地址
func (s *server) SetHost(host string) {
	s.config.Host = host
}

// SetPort 设置监听端口
func (s *server) SetPort(port int) {
	s.config.Port = port
}

// SetLogger 设置日志
func (s *server) SetLogger(logger zerologger.Logger) {
	s.config.Logger = logger
}

// SetLoggerLevel 设置日志级别
// 见 https://github.com/zerogo-hub/zero-helper/blob/main/logger/logger.go
func (s *server) SetLoggerLevel(loggerLevel int) {
	s.config.LoggerLevel = loggerLevel
}

// SetOnServerStart 服务器启动时触发，套接字监听此时尚未启动
func (s *server) SetOnServerStart(onServerStart func() error) {
	s.config.OnServerStart = onServerStart
}

// SetOnServerClose 服务端关闭时触发，此时已关闭客户端连接
func (s *server) SetOnServerClose(onServerClose func()) {
	s.config.OnServerClose = onServerClose
}

// SetCloseTimeout 关闭服务器的等待时间，超过该时间服务器直接关闭
func (s *server) SetCloseTimeout(closeTimeout time.Duration) {
	s.config.CloseTimeout = closeTimeout
}

// SetRecvBufferSize 在 session 中接收消息 buffer 大小
func (s *server) SetRecvBufferSize(recvBufferSize int) {
	s.config.RecvBufferSize = recvBufferSize
}

// SetRecvDeadline 通信超时时间，最终调用 conn.SetReadDeadline
func (s *server) SetRecvDeadline(recvDeadLine time.Duration) {
	s.config.RecvDeadline = recvDeadLine
}

// SetRecvQueueSize 在 session 中接收到的消息队列大小，session 接收到消息后并非立即处理，而是丢到一个消息队列中，异步处理
func (s *server) SetRecvQueueSize(recvQueueSize int) {
	s.config.RecvQueueSize = recvQueueSize
}

// SetSendBufferSize 发送消息 buffer 大小
func (s *server) SetSendBufferSize(recvBufferSize int) {
	s.config.RecvBufferSize = recvBufferSize
}

// SetSendDeadline SendDeadline
func (s *server) SetSendDeadline(recvDeadLine time.Duration) {
	s.config.RecvDeadline = recvDeadLine
}

// SetSendQueueSize 发送的消息队列大小，消息优先发送到 sesion 的消息队列，然后写入到套接字中
func (s *server) SetSendQueueSize(recvQueueSize int) {
	s.config.RecvQueueSize = recvQueueSize
}

// SetOnConnected 客户端连接到来时触发，此时客户端已经可以开始收发消息
func (s *server) SetOnConnected(onConnected zeronetwork.ConnFunc) {
	s.config.OnConnected = onConnected
}

// SetOnConnClose 客户端连接关闭触发，此时客户端不可以再收发消息
func (s *server) SetOnConnClose(onConnClose zeronetwork.ConnFunc) {
	s.config.OnConnClose = onConnClose
}

// SetDatapack 封包与解包
func (s *server) SetDatapack(datapack zeronetwork.Datapack) {
	s.config.Datapack = datapack
}

// SetWhetherCompress 是否需要对消息负载进行压缩
func (s *server) SetWhetherCompress(whetherCompress bool) {
	s.config.WhetherCompress = whetherCompress
}

// SetCompressThreshold 压缩的阈值，当消息负载长度超过该值时才会压缩
func (s *server) SetCompressThreshold(compressThreshold int) {
	s.config.CompressThreshold = compressThreshold
}

// SetCompress 压缩与解压器
func (s *server) SetCompress(compress zerocompress.Compress) {
	s.config.Compress = compress
}

// SetWhetherCrypto 是否需要对消息负载进行加密
func (s *server) SetWhetherCrypto(whetherCrypto bool) {
	s.config.WhetherCrypto = whetherCrypto
}

// SetWhetherChecksum 是否启用校验值功能，默认 false
func (s *server) SetWhetherChecksum(whetherChecksum bool) {
	s.config.WhetherChecksum = whetherChecksum
}

// listen 启动监听
func (s *server) listen() {
	address := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	ln, err := kcp.ListenWithOptions(address, nil, s.kcpConfig.datashard, s.kcpConfig.parityshard)
	if err != nil {
		s.config.Logger.Fatalf("net.ListenTCP error: %s, address: %s", err.Error(), address)
		return
	}

	// 异常退出
	defer func() {
		if p := recover(); p != nil {
			s.config.Logger.Errorf("recover error: %+v", p)
		}

		s.Close()

		s.config.Logger.Info("server close")
	}()

	s.ln = ln

	// 监听，开始 accept
	s.config.Logger.Infof("server start, listen at %s, pid: %d", address, os.Getpid())

	for {
		conn, err := ln.AcceptKCP()
		if err != nil {
			if s.isClosed {
				break
			}

			s.config.Logger.Error(err.Error())
			continue
		}

		remoteAddress := conn.RemoteAddr().String()

		// 服务器已经关闭
		if s.isClosed {
			conn.Close()
			s.Logger().Infof("reject conn, server is closed, remote remoteAddress: %s", remoteAddress)
			break
		}

		// 此时不接收新的连接
		if s.isCloseConn {
			conn.Close()
			s.Logger().Infof("reject conn, conn is closed, remote remoteAddress: %s", remoteAddress)
			continue
		}

		// 是否超出连接数量上限，关闭新的连接
		if s.config.MaxConnNum > 0 && s.sessionManager.Len() >= s.config.MaxConnNum {
			_ = conn.Close()
			s.Logger().Infof("reject conn, max conn num, remote remoteAddress: %s", remoteAddress)
			continue
		}

		conn.SetWindowSize(s.kcpConfig.sndwnd, s.kcpConfig.rcvwnd)
		conn.SetNoDelay(s.kcpConfig.nodelay, s.kcpConfig.interval, s.kcpConfig.resend, s.kcpConfig.nc)
		conn.SetStreamMode(s.kcpConfig.streamMode)
		conn.SetMtu(s.kcpConfig.mtu)
		if err := conn.SetReadBuffer(s.config.RecvBufferSize); err != nil {
			_ = conn.Close()
			s.Logger().Infof("conn SetReadBuffer failed, remote remoteAddress: %s, err: %s", remoteAddress, err.Error())
			continue
		}
		if err := conn.SetWriteBuffer(s.config.SendBufferSize); err != nil {
			_ = conn.Close()
			s.Logger().Infof("conn SetWriteBuffer failed, remote remoteAddress: %s, err: %s", remoteAddress, err.Error())
			continue
		}

		// session 用于管理该连接
		session := newSession(
			s.sessionManager.GenSessionID(),
			conn,
			s.config,
			s.closeSession,
			s.router.Handler,
		)
		s.sessionManager.Add(session)
		s.Logger().Infof("session: %d, address: %s connected", session.ID(), remoteAddress)

		go session.Run()
	}
}

// closeSession 关闭会话后的回调
func (s *server) closeSession(session zeronetwork.Session) {
	s.sessionManager.Del(session.ID())
}

// ListenSignal 监听信号
func (s *server) ListenSignal() {
	// ctrl + c 或者 kill
	sigs := []os.Signal{syscall.SIGINT, syscall.SIGTERM}

	ch := make(chan os.Signal, 1)

	signal.Notify(ch, sigs...)

	sig := <-ch

	signal.Stop(ch)

	s.config.Logger.Infof("received signal, sig: %+v", sig)

	// 关闭服务器
	s.Close()
}
