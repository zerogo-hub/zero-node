package tcp

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"

	zerologger "github.com/zerogo-hub/zero-helper/logger"
	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
)

// server tcp 服务
// 实现接口: Peer
type server struct {
	config *Config

	// ln 监听套接字
	ln *net.TCPListener

	// sessionManager 会话管理
	sessionManager zeronetwork.SessionManager

	// genSessionID 用于生成会话 ID
	genSessionID zeronetwork.SessionID

	// closeOnce 防止多次关闭服务
	closeOnce sync.Once

	// isClosed 服务器已关闭
	isClosed bool

	// isCloseConn 服务器不再接收连接
	isCloseConn bool

	// router 路由
	router zeronetwork.Router
}

// NewServer 创建一个 tcp 服务
func NewServer(opts ...Option) zeronetwork.Peer {
	s := &server{
		config:         defaultConfig(),
		sessionManager: newSessionManager(),
		router:         zeronetwork.NewRouter(),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Start 开启服务
func (s *server) Start() error {
	if err := s.config.onServerStart(); err != nil {
		return err
	}

	go s.listen()

	s.signal()
	return nil
}

// Logger 日志
func (s *server) Logger() zerologger.Logger {
	return s.config.logger
}

// Router 路由器
func (s *server) Router() zeronetwork.Router {
	return s.router
}

// listen 启动监听
func (s *server) listen() {
	address := fmt.Sprintf("%s:%d", s.config.host, s.config.port)
	addr, err := net.ResolveTCPAddr(s.config.network, address)
	if err != nil {
		s.config.logger.Fatalf("net.ResolveTCPAddr error: %s, network: %s, address: %s", err.Error(), s.config.network, address)
		return
	}

	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		s.config.logger.Fatalf("net.ListenTCP error: %s, network: %s, address: %s", err.Error(), s.config.network, address)
		return
	}

	// 异常退出
	defer func() {
		if p := recover(); p != nil {
			s.config.logger.Errorf("recover error: %+v", p)
		}

		s.Close()

		s.config.logger.Info("server close")
	}()

	s.ln = ln

	// 监听，开始 accept
	s.config.logger.Infof("server start, listen at %s, pid: %d", address, os.Getpid())

	for {
		conn, err := ln.AcceptTCP()
		if err != nil {
			if s.isClosed {
				break
			}

			s.config.logger.Error(err.Error())
			continue
		}

		// 服务器已经关闭
		if s.isClosed {
			conn.Close()
			break
		}

		// 此时不接收新的连接
		if s.isCloseConn {
			conn.Close()
			continue
		}

		// 是否超出连接数量上限，关闭新的连接
		if s.config.maxConnNum > 0 && s.sessionManager.Len() >= s.config.maxConnNum {
			conn.Close()
			continue
		}

		// 设置连接属性
		conn.SetKeepAlive(true)
		conn.SetNoDelay(true)
		conn.SetReadBuffer(s.config.recvBufferSize)
		conn.SetWriteBuffer(s.config.sendBufferSize)

		// session 用于管理该连接
		atomic.AddUint64(&s.genSessionID, 1)
		session := newSession(s.genSessionID, conn, s.config, s.router.Handler)
		s.sessionManager.Add(session)

		go session.Run()
	}
}

// kickout 主动断开该会话
func (s *server) kickout(sessionID zeronetwork.SessionID) {
	s.sessionManager.Del(sessionID)
}

// signal 监听信号
func (s *server) signal() {
	// ctrl + c 或者 kill
	sigs := []os.Signal{syscall.SIGINT, syscall.SIGTERM}

	ch := make(chan os.Signal)

	signal.Notify(ch, sigs...)

	sig := <-ch

	signal.Stop(ch)

	s.config.logger.Infof("received signal, sig: %+v", sig)

	// 关闭服务器
	s.Close()
}

// Close 关闭服务，释放资源
func (s *server) Close() error {
	s.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), s.config.closeTimeout)
		defer cancel()

		ch := make(chan bool)

		go func() {
			s.isClosed = true
			s.isCloseConn = true

			// 停止监听
			if err := s.ln.Close(); err != nil {
				s.config.logger.Errorf("close listen failed: %s", err.Error())
			}

			// 关闭所有连接
			s.sessionManager.Close()

			// 处理自定义行为
			if s.config.onServerClose != nil {
				s.config.onServerClose()
			}

			ch <- true
		}()

		select {
		case <-ch:
			s.config.logger.Info("close success")
			break
		case <-ctx.Done():
			s.config.logger.Error("close timeout")
			break
		}
	})

	return nil
}
