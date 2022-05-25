package tcp

import (
	"fmt"

	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
)

// messageHead 消息头
// 消息头长度 10
type messageHead struct {
	// Len 负载长度，即 Payload 中的长度
	Len uint16
	// Flag 标记，具体见 modules/network/flag.go
	Flag uint16
	// SN 自增编号，由客户端发出，服务端原样返回。服务端主动发出的消息中 SN 值为 0
	SN uint16
	// Code 错误码，如果存在错误，则会在 payload 中存储具体的错误信息
	Code uint16
	// Module 功能模块，用来表示一个功能大类，比如商店、副本
	Module uint8
	// Action 功能细分，用来表示一个功能里面的具体功能，比如进入副本，退出副本
	Action uint8
}

// HeadLen 消息头长度
func HeadLen() int {
	return 10
}

// message 消息
type message struct {
	// Head 消息头
	head *messageHead
	// Payload 具体内容
	payload []byte

	// SessionID 会话 id
	sessionID zeronetwork.SessionID
}

// NewMessage 创建一个消息
func NewMessage(flag, sn, code uint16, module, action uint8, payload []byte) zeronetwork.Message {
	return &message{
		head: &messageHead{
			Len:    uint16(len(payload)),
			Flag:   flag,
			SN:     sn,
			Code:   code,
			Module: module,
			Action: action,
		},
		payload: payload,
	}
}

// SessionID 会话 ID，每一个连接都有一个唯一的会话 ID
func (m *message) SessionID() zeronetwork.SessionID {
	return m.sessionID
}

// SetSessionID 设置 sessionID
func (m *message) SetSessionID(sessionID zeronetwork.SessionID) {
	m.sessionID = sessionID
}

// Code 错误码
func (m *message) Code() uint16 {
	return m.head.Code
}

// ModuleID 功能模块，用来表示一个功能大类，比如商店、副本
func (m *message) ModuleID() uint8 {
	return m.head.Module
}

// ActionID 功能细分，用来表示一个功能里面的具体功能，比如进入副本，退出副本
func (m *message) ActionID() uint8 {
	return m.head.Action
}

// Flag 标记
func (m *message) Flag() uint16 {
	return m.head.Flag
}

// SN 自增编号
func (m *message) SN() uint16 {
	return m.head.SN
}

// Payload 负载
func (m *message) Payload() []byte {
	return m.payload
}

// String 打印信息
func (m *message) String() string {
	return fmt.Sprintf("sn: %d, module: %d, action: %d", m.head.SN, m.head.Module, m.head.Action)
}
