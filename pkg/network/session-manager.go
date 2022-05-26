package network

import (
	"errors"
	"sync"
)

var (
	// ErrSessionNotFound Session 未找到
	ErrSessionNotFound = errors.New("session not found")
)

// sessionManager 会话管理器，实现 network.go/SessionManager 接口
type sessionManager struct {
	// sessions 存储所有连接
	sessions map[SessionID]Session

	// lock 读写锁，保护 sessions
	lock sync.RWMutex
}

// NewSessionManager 创建会话管理器
func NewSessionManager() SessionManager {
	return &sessionManager{
		sessions: make(map[SessionID]Session, 8192),
	}
}

// Add 添加 Session
func (m *sessionManager) Add(session Session) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.sessions[session.ID()] = session
}

// Del 移除 Session
func (m *sessionManager) Del(sessionID SessionID) {
	m.lock.Lock()
	defer m.lock.Unlock()

	if session, ok := m.sessions[sessionID]; ok {
		session.Close()

		delete(m.sessions, sessionID)
	}
}

// Get(sessionID SessionID) (Session, error)
func (m *sessionManager) Get(sessionID SessionID) (Session, error) {
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
	m.sessions = make(map[SessionID]Session)
}

// Send 发送消息给客户端
func (m *sessionManager) Send(sessionID SessionID, message Message) error {
	session, err := m.Get(sessionID)
	if err != nil {
		return err
	}

	return session.Send(message)
}

// SendCallback  发送消息个客户端，发送之后进行回调
func (m *sessionManager) SendCallback(sessionID SessionID, message Message, callback SendCallbackFunc) error {
	session, err := m.Get(sessionID)
	if err != nil {
		return err
	}

	return session.SendCallback(message, callback)
}

// SendAll 给所有客户端发送消息
func (m *sessionManager) SendAll(message Message) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	for _, session := range m.sessions {
		session.Send(message)
	}
}
