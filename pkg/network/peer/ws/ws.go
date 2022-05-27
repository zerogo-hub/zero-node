package ws

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	websocket "github.com/gorilla/websocket"
	zerocompress "github.com/zerogo-hub/zero-helper/compress"
	zerologger "github.com/zerogo-hub/zero-helper/logger"
	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
	zerodatapack "github.com/zerogo-hub/zero-node/pkg/network/datapack"
)

var (
	upgrader = websocket.Upgrader{
		// 允许跨域
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// server websocket 服务
type server struct {
	config *zeronetwork.Config

	// sessionManager 会话管理
	sessionManager zeronetwork.SessionManager

	// closeOnce 防止多次关闭服务
	closeOnce sync.Once

	// isClosed 服务器已关闭
	isClosed bool

	// isCloseConn 服务器不再接收连接
	isCloseConn bool

	// router 路由
	router zeronetwork.Router

	// messageType 消息类型
	// 见 github.com/gorilla/websocket/conn.go 中的定义
	// 如 TextMessage、BinaryMessage
	messageType int

	certFile, keyFile string
}

// NewServer 创建一个 websocket 服务
func NewServer(messageType int, certFile, keyFile string, opts ...zeronetwork.Option) zeronetwork.Peer {
	s := &server{
		config:         zeronetwork.DefaultConfig(),
		sessionManager: zeronetwork.NewSessionManager(),
		router:         zeronetwork.NewRouter(),
		messageType:    messageType,
		certFile:       certFile,
		keyFile:        keyFile,
	}

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
	address := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	go func() {
		if len(s.certFile) > 0 && len(s.keyFile) > 0 {
			s.config.Logger.Infof("certFile: %s, keyFile: %s", s.certFile, s.keyFile)
			if err := http.ListenAndServeTLS(address, s.certFile, s.keyFile, nil); err != nil {
				s.Logger().Errorf("listen failed, address: %s, err: %s", address, err.Error())
			}
		} else {
			if err := http.ListenAndServe(address, nil); err != nil {
				s.Logger().Errorf("listen failed, address: %s, err: %s", address, err.Error())
			}
		}
	}()

	s.config.Logger.Infof("server start, listen at %s, pid: %d", address, os.Getpid())
	if err := s.config.OnServerStart(); err != nil {
		return err
	}
	http.HandleFunc("/", s.wsHandler)

	s.signal()
	return nil
}

// Close 关闭服务，释放资源
func (s *server) Close() error {
	if s.isClosed {
		return nil
	}

	s.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), s.config.CloseTimeout)
		defer cancel()

		ch := make(chan bool)

		go func() {
			s.isClosed = true
			s.isCloseConn = true

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
	})

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
	s.config.Network = network
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

// SetRecvDeadLine 通信超时时间，最终调用 conn.SetReadDeadline
func (s *server) SetRecvDeadLine(recvDeadLine time.Duration) {
	s.config.RecvDeadLine = recvDeadLine
}

// SetRecvQueueSize 在 session 中接收到的消息队列大小，session 接收到消息后并非立即处理，而是丢到一个消息队列中，异步处理
func (s *server) SetRecvQueueSize(recvQueueSize int) {
	s.config.RecvQueueSize = recvQueueSize
}

// SetSendBufferSize 发送消息 buffer 大小
func (s *server) SetSendBufferSize(recvBufferSize int) {
	s.config.RecvBufferSize = recvBufferSize
}

// SetSendDeadLine SendDeadline
func (s *server) SetSendDeadLine(recvDeadLine time.Duration) {
	s.config.RecvDeadLine = recvDeadLine
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

// wsHandler 客户端连接过来时的处理
// 将原本的 http 请求升级为 websocket
func (s *server) wsHandler(w http.ResponseWriter, r *http.Request) {
	// 服务器已经关闭
	if s.isClosed {
		return
	}
	// 此时不接收新的连接
	if s.isCloseConn {
		return
	}

	// 是否超出连接数量上限，关闭新的连接
	if s.config.MaxConnNum > 0 && s.sessionManager.Len() >= s.config.MaxConnNum {
		return
	}

	// 完成 websocket 协议的握手操作
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	// session 用于管理该连接
	session := newSession(
		s.sessionManager.GenSessionID(),
		conn,
		s.config,
		s.closeSession,
		s.router.Handler,
		s.messageType,
	)
	s.sessionManager.Add(session)

	go session.Run()
}

// signal 监听信号
func (s *server) signal() {
	// ctrl + c 或者 kill
	sigs := []os.Signal{syscall.SIGINT, syscall.SIGTERM}

	ch := make(chan os.Signal)

	signal.Notify(ch, sigs...)

	sig := <-ch

	signal.Stop(ch)

	s.config.Logger.Infof("received signal, sig: %+v", sig)

	// 关闭服务器
	s.Close()
}

// closeSession 关闭会话后的回调
func (s *server) closeSession(session zeronetwork.Session) {
	s.sessionManager.Del(session.ID())
}
