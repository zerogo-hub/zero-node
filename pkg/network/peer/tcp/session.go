package tcp

import (
	"errors"
	"io"
	"net"
	"sync"
	"time"

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

	// ErrSessionNotFound Session 未找到
	ErrSessionNotFound = errors.New("session not found")
)

// session 会话，实现 network.go/Session 接口
// 一个会话会开启 3 个 goroutine
// 1: sendLoop
// 2: recvLoop
// 3: dispatchLoop
// 收到的消息会从 recvLoop 中放入到 recvQueue
// dispatchLoop 会处理 recvQueue 消息
// 处理之后会将要发送的消息放入到 sendQueue
type session struct {
	// config 一些通用配置
	config *Config

	// sessionID 会话 ID，每一条链接都有一个唯一的 ID
	sessionID zeronetwork.SessionID

	// conn 客户端与服务器链接成功后的原始套接字，由 Accept() 生成
	conn *net.TCPConn

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

	// crypto 消息负载的加密与解密
	crypto zeronetwork.Crypto

	// handler 用于处理接收到的消息
	handler zeronetwork.HandlerFunc
}

// sendElement 表示一个将要发送的消息
type sendElement struct {
	// message 将要发送的网络消息
	message zeronetwork.Message
	// callback 发送成功之后的回调
	callback zeronetwork.SendCallbackFunc
}

// newSession 创建一个 tcp 会话
func newSession(sessionID zeronetwork.SessionID, conn *net.TCPConn, config *Config, handler zeronetwork.HandlerFunc) zeronetwork.Session {
	session := &session{
		config:    config,
		sessionID: sessionID,
		conn:      conn,
		sendQueue: make(chan *sendElement, config.sendQueueSize),
		recvQueue: make(chan zeronetwork.Message, config.recvQueueSize),
		closeCh:   make(chan bool),
		handler:   handler,
	}

	return session
}

// Run 让当前连接开始工作，比如收发消息，一般用于连接成功之后
func (s *session) Run() {
	if s.config.onConnected != nil {
		s.config.onConnected(s)
	}

	go s.recvLoop()
	go s.dispatchLoop()
	s.sendLoop()
}

// Close 停止接收客户端消息，也不再接收服务端消息。当已接收的服务端消息发送完毕后，断开连接
func (s *session) Close() {
	s.closeOnce.Do(func() {
		defer func() {
			if p := recover(); p != nil {
				s.config.logger.Errorf("session: %d close, recover error: %s", s.ID(), p)
			}

			if s.config.logger.IsDebugAble() {
				s.config.logger.Debugf("session: %d closed", s.ID())
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
		// 7 执行关闭时的触发函数
		if s.config.onConnClose != nil {
			s.config.onConnClose(s)
		}

		s.config.logger.Infof("session: %d closed", s.ID())
	})
}

// Send 发送消息给客户端
func (s *session) Send(message zeronetwork.Message) error {
	return s.SendCallback(message, nil)
}

// SendCallback 发送消息给客户端，发送之后还有回调函数
func (s *session) SendCallback(message zeronetwork.Message, callback zeronetwork.SendCallbackFunc) error {
	if s.isStopSend {
		// 不再发送新的消息
		return ErrStopSend
	}

	// 发送发送队列，异步发送
	select {
	case s.sendQueue <- &sendElement{message: message, callback: callback}:
		if s.config.logger.IsDebugAble() {
			s.config.logger.Debugf("session: %d, send to queue success, message: %s", s.ID(), message.String())
		}
		return nil
	case <-time.After(3 * time.Second):
		s.config.logger.Errorf("session: %d, send to queue timeout, message: %s", s.ID(), message.String())
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
	return s.conn
}

// SetCrypto 设置加密解密的工具
func (s *session) SetCrypto(crypto zeronetwork.Crypto) {
	s.crypto = crypto
}

// recvLoop 接收消息
func (s *session) recvLoop() {
	defer func() {
		if p := recover(); p != nil {
			s.config.logger.Errorf("recover p: %+v", p)
		}

		s.Close()
	}()

	headerLen := s.config.datapack.HeadLen()

	recvBufferSize := s.config.recvBufferSize

	// buffer 用于读取 socket 中的数据
	buffer := make([]byte, recvBufferSize)

	// circleBuffer 用于存储从 socket 读取的数据
	circleBuffer := zerocircle.New(recvBufferSize * 2)
	circleBuffer.Reset()

	for {
		if s.config.recvDeadLine > 0 {
			if err := s.conn.SetReadDeadline(time.Now().Add(s.config.recvDeadLine)); err != nil {
				s.config.logger.Error("session: %d, set read deadline error: %s, deadline: %d", s.ID(), err.Error(), s.config.recvDeadLine)
				break
			}
		}

		size, err := io.ReadAtLeast(s.conn, buffer, headerLen)

		if s.isStopRecv {
			break
		}

		if err != nil {
			// 远端关闭
			if err == io.EOF {
				if s.config.logger.IsDebugAble() {
					s.config.logger.Debugf("session: %d, closed by remote, io.EOF", s.ID())
				}
			} else {
				s.config.logger.Errorf("session: %d, read failed: %s", s.ID(), err.Error())
			}
			break
		}

		if size == 0 {
			if s.config.logger.IsDebugAble() {
				s.config.logger.Debugf("session: %d closed by remote, size is zero", s.ID())
			}
			break
		}

		// 在 circleBuffer 中存储所有收到的消息
		// 需要注意的是，尚未处理的消息 + 收到的 buffer 的长度不得超过 circleBuffer 的长度
		err = circleBuffer.WriteN(buffer, size)
		if err != nil {
			s.config.logger.Errorf("session: %d, write to circle buffer failed: %s", s.ID(), err.Error())
			break
		}

		messages, err := s.config.datapack.Unpack(circleBuffer, s.crypto)
		if err != nil {
			s.config.logger.Errorf("session: %d unpack failed: %s", s.ID(), err.Error())
			break
		}

		if len(messages) > 0 {
			if err := s.dispatch(messages); err != nil {
				s.config.logger.Errorf("session: %d dispatch failed: %s", s.ID(), err.Error())
				break
			}
		}
	}
}

// dispatch 将接收到的客户端消息进行处理
func (s *session) dispatch(messages []zeronetwork.Message) error {
	for _, message := range messages {
		// 消息设置连接 ID
		message.SetSessionID(s.sessionID)

		s.recvQueue <- message
	}

	return nil
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
				if s.config.logger.IsDebugAble() {
					s.config.logger.Debugf("session: %d, dispatch message failed: %s, message: %s", message.SessionID(), err.Error(), message.String())
				}
				// s.kickout(message.SessionID())
				break
			}

			if responseMessage != nil {
				if err := s.Send(responseMessage); err != nil {
					s.config.logger.Errorf("session: %d, send response message failed: %s, message: %s", message.SessionID(), err.Error(), message.String())
					// s.kickout(message.SessionID())
					break
				}
			}
		}
	}
}

// sendLoop 发送消息
func (s *session) sendLoop() {
	defer func() {
		if p := recover(); p != nil {
			s.config.logger.Errorf("recover p: %+v", p)
		}

		s.Close()
	}()

	for {
		select {
		case element, ok := <-s.sendQueue:
			if !ok {
				s.config.logger.Errorf("session: %d, sendQueue error", s.ID())
				return
			}

			if err := s.write(element.message); err != nil {
				s.config.logger.Errorf("session: %d, message: %s, write failed: %s", s.ID(), element.message.String(), err.Error())
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

	if s.config.sendDeadLine > 0 {
		if err := s.conn.SetWriteDeadline(time.Now().Add(s.config.sendDeadLine)); err != nil {
			s.config.logger.Errorf("session: %d, set write deadline failed: %s, deadline: %d", s.ID, err.Error(), s.config.sendDeadLine)
			return err
		}
	}

	p, err := s.config.datapack.Pack(message, s.crypto)
	if err != nil {
		s.config.logger.Errorf("session: %d, pack message failed; %s, message: %s", s.ID, err.Error(), message.String())
		return err
	}

	n, err := s.conn.Write(p)
	if err != nil {
		s.config.logger.Errorf("session: %d, conn write failed: %s, message: %s", s.ID, err.Error(), message.String())
		return err
	}

	if n != len(p) {
		s.config.logger.Errorf("session: %d, write data is not complete: %d/%d", n, len(p))
		return ErrWriteNotAll
	}

	return nil
}

// sessionManager 会话管理器，实现 network.go/SessionManager 接口
type sessionManager struct {
	// sessions 存储所有连接
	sessions map[zeronetwork.SessionID]zeronetwork.Session

	// lock 读写锁，保护 sessions
	lock sync.RWMutex
}

// newSessionManager 创建会话管理器
func newSessionManager() zeronetwork.SessionManager {
	return &sessionManager{
		sessions: make(map[zeronetwork.SessionID]zeronetwork.Session, 8192),
	}
}

// Add 添加 Session
func (m *sessionManager) Add(session zeronetwork.Session) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.sessions[session.ID()] = session
}

// Del 移除 Session
func (m *sessionManager) Del(sessionID zeronetwork.SessionID) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if session, ok := m.sessions[sessionID]; ok {
		session.Close()

		delete(m.sessions, sessionID)
	}
}

// Get(sessionID SessionID) (Session, error)
func (m *sessionManager) Get(sessionID zeronetwork.SessionID) (zeronetwork.Session, error) {
	if session, ok := m.sessions[sessionID]; ok {
		return session, nil
	}

	return nil, ErrSessionNotFound
}

// Len 获取当前 Session 数量
func (m *sessionManager) Len() int {
	m.lock.RLock()
	defer m.lock.RUnlock()

	return len(m.sessions)
}

// Close 当前所有连接停止接收客户端消息，不再接收服务端消息，当已接收的服务端消息发送完毕后，断开连接
// timeout 超时时间，如果超时仍未发送完已接收的服务端消息，也强行关闭连接
func (m *sessionManager) Close() {
	m.lock.Lock()
	defer m.lock.Unlock()

	for _, session := range m.sessions {
		session.Close()
	}

	// 清空
	m.sessions = make(map[zeronetwork.SessionID]zeronetwork.Session)
}

// Send 发送消息给客户端
func (m *sessionManager) Send(sessionID zeronetwork.SessionID, message zeronetwork.Message) error {
	session, err := m.Get(sessionID)
	if err != nil {
		return err
	}

	return session.Send(message)
}

// SendCallback  发送消息个客户端，发送之后进行回调
func (m *sessionManager) SendCallback(sessionID zeronetwork.SessionID, message zeronetwork.Message, callback zeronetwork.SendCallbackFunc) error {
	session, err := m.Get(sessionID)
	if err != nil {
		return err
	}

	return session.SendCallback(message, callback)
}

// SendAll 给所有客户端发送消息
func (m *sessionManager) SendAll(message zeronetwork.Message) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, session := range m.sessions {
		session.Send(message)
	}
}
