package network

import (
	"errors"
	"sync"
	"sync/atomic"
)

var (
	// ErrSessionNotFound Session 未找到
	ErrSessionNotFound = errors.New("session not found")
)

// sessionManager 会话管理器，实现 network.go/SessionManager 接口
type sessionManager struct {
	// sessions 存储所有连接
	sessions map[SessionID]Session

	// genSessionID 用于生成会话 ID
	genSessionID SessionID

	// lock 读写锁，保护 sessions
	lock sync.RWMutex
}

// NewSessionManager 创建会话管理器
func NewSessionManager() SessionManager {
	return &sessionManager{
		sessions: make(map[SessionID]Session, 8192),
	}
}

// GenSessionID 生成新的会话 ID
func (s *sessionManager) GenSessionID() SessionID {
	return atomic.AddUint64(&s.genSessionID, 1)
}

// Add 添加 Session
func (s *sessionManager) Add(session Session) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.sessions[session.ID()] = session
}

// Del 移除 Session
func (s *sessionManager) Del(sessionID SessionID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if session, ok := s.sessions[sessionID]; ok {
		session.Close()

		delete(s.sessions, sessionID)
	}
}

// Get(sessionID SessionID) (Session, error)
func (s *sessionManager) Get(sessionID SessionID) (Session, error) {
	if session, ok := s.sessions[sessionID]; ok {
		return session, nil
	}

	return nil, ErrSessionNotFound
}

// Len 获取当前 Session 数量
func (s *sessionManager) Len() int {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return len(s.sessions)
}

// Close 当前所有连接停止接收客户端消息，不再接收服务端消息，当已接收的服务端消息发送完毕后，断开连接
// timeout 超时时间，如果超时仍未发送完已接收的服务端消息，也强行关闭连接
func (s *sessionManager) Close() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, session := range s.sessions {
		session.Close()
	}

	// 清空
	s.sessions = make(map[SessionID]Session)
}

// Send 发送消息给客户端
func (s *sessionManager) Send(sessionID SessionID, message Message) error {
	session, err := s.Get(sessionID)
	if err != nil {
		return err
	}

	return session.Send(message)
}

// SendCallback  发送消息个客户端，发送之后进行回调
func (s *sessionManager) SendCallback(sessionID SessionID, message Message, callback SendCallbackFunc) error {
	session, err := s.Get(sessionID)
	if err != nil {
		return err
	}

	return session.SendCallback(message, callback)
}

// SendAll 给所有客户端发送消息
func (s *sessionManager) SendAll(message Message) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, session := range s.sessions {
		session.Send(message)
	}
}
