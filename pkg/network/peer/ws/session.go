package ws

import (
	"errors"
	"net"
	"sync"
	"time"

	websocket "github.com/gorilla/websocket"
	zerocircle "github.com/zerogo-hub/zero-helper/buffer/circle"
	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
)

var (
	// ErrWriteNotAll 未能将信息全部写入
	ErrWriteNotAll = errors.New("write not all")

	// ErrStopSend 已关闭，不再收发消息
	ErrStopSend = errors.New("stop send message")

	// ErrWriteTimeout 放入发送队列超时 3秒
	ErrWriteTimeout = errors.New("write timeout")
)

// session 会话，实现 network.go/Session 接口
// 一个会话会开启 3 个 goroutine
// 1: sendLoop
// 2: recvLoop
// 3: dispatchLoop
// 收到客户端的消息会从 recvLoop 中放入到 recvQueue
// dispatchLoop 会处理 recvQueue 消息
// 处理之后会将要发送的消息放入到 sendQueue
// 服务端主动推送的消息也会放到 sendQueue
// sendLoop 会将放在 sendQueue 中的消息发送到客户端
type session struct {
	// config 一些通用配置
	config *zeronetwork.Config

	// sessionID 会话 ID，每一条链接都有一个唯一的 ID
	sessionID zeronetwork.SessionID

	// conn gorilla/websocket 的 Conn
	conn *websocket.Conn

	// closeOnce 防止多次关闭会话
	closeOnce sync.Once

	// isStopRecv 是否停止接收消息
	isStopRecv bool

	// isStopSend 是否停止发送消息
	isStopSend bool

	// sendQueue 发送消息队列
	sendQueue chan *sendElement

	// sendWait 用于保证消息全部发送完成
	sendWait sync.WaitGroup

	// recvQueue 存储接收到的消息
	recvQueue chan zeronetwork.Message

	// closeCh 关闭会话的信号
	closeCh chan bool

	// closeCallback 关闭会话后的回调
	closeCallback zeronetwork.CloseCallbackFunc

	// crypto 消息负载的加密与解密
	crypto zeronetwork.Crypto

	// handler 用于处理接收到的消息
	handler zeronetwork.HandlerFunc

	// messageType 在 gorilla/websocket 中定义的消息类型
	messageType int
}

// sendElement 表示一个将要发送的消息
type sendElement struct {
	// message 将要发送的网络消息
	message zeronetwork.Message
	// callback 发送成功之后的回调
	callback zeronetwork.SendCallbackFunc
}

// newSession 创建一个 ws 会话
func newSession(
	sessionID zeronetwork.SessionID,
	conn *websocket.Conn,
	config *zeronetwork.Config,
	closeCallback zeronetwork.CloseCallbackFunc,
	handler zeronetwork.HandlerFunc,
	messageType int,
) *session {
	session := &session{
		config:        config,
		sessionID:     sessionID,
		conn:          conn,
		sendQueue:     make(chan *sendElement, config.SendQueueSize),
		recvQueue:     make(chan zeronetwork.Message, config.RecvQueueSize),
		closeCh:       make(chan bool),
		closeCallback: closeCallback,
		handler:       handler,
		messageType:   messageType,
	}

	return session
}

// Run 让当前连接开始工作，比如收发消息，一般用于连接成功之后
func (s *session) Run() {
	if s.config.OnConnected != nil {
		s.config.OnConnected(s)
	}

	go s.recvLoop()
	go s.dispatchLoop()
	s.sendLoop()
}

// Close 停止接收客户端消息，也不再接收服务端消息。当已接收的服务端消息发送完毕后，断开连接
func (s *session) Close() {
	if s.isStopRecv && s.isStopSend {
		return
	}

	s.closeOnce.Do(func() {
		defer func() {
			if p := recover(); p != nil {
				s.config.Logger.Errorf("session: %d close, recover error: %s", s.ID(), p)
			}

			if s.config.Logger.IsDebugAble() {
				s.config.Logger.Debugf("session: %d closed", s.ID())
			}
		}()

		// 1 停止接收来自客户端的消息
		s.isStopRecv = true
		// 2 停止发送来自服务端的消息
		s.isStopSend = true
		// 3 等待发送队列中的消息发送完毕
		s.sendWait.Wait()
		// 4 关闭接收与发送循环
		s.closeCh <- true
		// 5 关闭套接字连接
		s.conn.Close()
		// 6 关闭所有通道
		close(s.closeCh)
		close(s.sendQueue)
		close(s.recvQueue)
		// 7 关闭会话后的回调
		if s.closeCallback != nil {
			s.closeCallback(s)
		}
		// 8 执行关闭时的触发函数
		if s.config.OnConnClose != nil {
			s.config.OnConnClose(s)
		}

		s.config.Logger.Infof("session: %d closed", s.ID())
	})
}

// Send 发送消息给客户端
func (s *session) Send(message zeronetwork.Message) error {
	return s.SendCallback(message, nil)
}

// SendCallback 发送消息给客户端，发送之后响应回调函数
func (s *session) SendCallback(message zeronetwork.Message, callback zeronetwork.SendCallbackFunc) error {
	if s.isStopSend {
		// 不再发送新的消息
		return ErrStopSend
	}

	// 发送发送队列，异步发送
	select {
	case s.sendQueue <- &sendElement{message: message, callback: callback}:
		if s.config.Logger.IsDebugAble() {
			s.config.Logger.Debugf("session: %d, send to queue success, message: %s", s.ID(), message.String())
		}
		return nil
	case <-time.After(3 * time.Second):
		s.config.Logger.Errorf("session: %d, send to queue timeout, message: %s", s.ID(), message.String())
		return ErrWriteTimeout
	}
}

// ID 获取 sessionID，每一条连接都分配有一个唯一的 id
func (s *session) ID() zeronetwork.SessionID {
	return s.sessionID
}

// RemoteAddr 客户端地址信息
func (s *session) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}

// Conn 获取原始的连接
func (s *session) Conn() net.Conn {
	return s.conn.UnderlyingConn()
}

// SetConn 设置原始的链接
func (s *session) SetConn(conn net.Conn) {
	panic("unimplement SetConn")
}

// SetCrypto 设置加密解密的工具
func (s *session) SetCrypto(crypto zeronetwork.Crypto) {
	s.crypto = crypto
}

// Config 配置
func (s *session) Config() *zeronetwork.Config {
	return s.config
}

func (s *session) recvLoop() {
	defer func() {
		if p := recover(); p != nil {
			s.config.Logger.Errorf("recover p: %+v", p)
		}

		s.Close()
	}()

	recvBufferSize := s.config.RecvBufferSize

	// circleBuffer 用于存储从 socket 读取的数据
	circleBuffer := zerocircle.New(recvBufferSize * 2)
	circleBuffer.Reset()

	var buffer []byte
	var err error

	for {
		if s.config.RecvDeadLine > 0 {
			if err := s.conn.SetReadDeadline(time.Now().Add(s.config.RecvDeadLine)); err != nil {
				s.config.Logger.Error("session: %d, set read deadline error: %s, deadline: %d", s.ID(), err.Error(), s.config.RecvDeadLine)
				break
			}
		}

		_, buffer, err = s.conn.ReadMessage()
		if err != nil {
			break
		}

		// 在 circleBuffer 中存储所有收到的消息
		// 需要注意的是，尚未处理的消息 + 收到的 buffer 的长度不得超过 circleBuffer 的长度
		err = circleBuffer.WriteN(buffer, len(buffer))
		if err != nil {
			s.config.Logger.Errorf("session: %d, write to circle buffer failed: %s", s.ID(), err.Error())
			break
		}

		messages, err := s.config.Datapack.Unpack(circleBuffer, s.crypto)
		if err != nil {
			s.config.Logger.Errorf("session: %d unpack failed: %s", s.ID(), err.Error())
			break
		}

		// 将消息存入缓冲队列 recvQueue 中，等待 dispatchLoop 处理
		for _, message := range messages {
			// 消息设置连接 ID
			message.SetSessionID(s.sessionID)

			s.recvQueue <- message
		}
	}
}

// dispatchLoop 执行 recvQueue 中的消息
func (s *session) dispatchLoop() {
	defer s.Close()

	for {
		select {
		case message, ok := <-s.recvQueue:
			if !ok {
				break
			}

			responseMessage, err := s.handler(message)
			if err != nil {
				if s.config.Logger.IsDebugAble() {
					s.config.Logger.Debugf("session: %d, dispatch message failed: %s, message: %s", message.SessionID(), err.Error(), message.String())
				}
				break
			}

			if responseMessage != nil {
				if err := s.Send(responseMessage); err != nil {
					s.config.Logger.Errorf("session: %d, send response message failed: %s, message: %s", message.SessionID(), err.Error(), message.String())
					break
				}
			}
		case <-s.closeCh:
			return
		}
	}
}

func (s *session) sendLoop() {
	defer func() {
		if p := recover(); p != nil {
			s.config.Logger.Errorf("recover p: %+v", p)
		}

		s.Close()
	}()

	for {
		select {
		case element, ok := <-s.sendQueue:
			if !ok {
				s.config.Logger.Errorf("session: %d, sendQueue error", s.ID())
				return
			}

			if err := s.write(element.message); err != nil {
				s.config.Logger.Errorf("session: %d, message: %s, write failed: %s", s.ID(), element.message.String(), err.Error())
				return
			}

			if element.callback != nil {
				element.callback(s)
			}
		case <-s.closeCh:
			return
		}
	}
}

// write 将消息写入套接字
func (s *session) write(message zeronetwork.Message) error {
	s.sendWait.Add(1)
	defer s.sendWait.Done()

	p, err := s.config.Datapack.Pack(message, s.crypto)
	if err != nil {
		s.config.Logger.Errorf("session: %d, pack message failed; %s, message: %s", s.ID, err.Error(), message.String())
		return err
	}

	if s.config.SendDeadLine > 0 {
		if err := s.conn.SetWriteDeadline(time.Now().Add(s.config.SendDeadLine)); err != nil {
			s.config.Logger.Errorf("session: %d, set write deadline failed: %s, deadline: %d", s.ID, err.Error(), s.config.SendDeadLine)
			return err
		}
	}

	err = s.conn.WriteMessage(s.messageType, p)
	if err != nil {
		s.config.Logger.Errorf("session: %d, conn write failed: %s, message: %s", s.ID, err.Error(), message.String())
		return err
	}

	return nil
}
