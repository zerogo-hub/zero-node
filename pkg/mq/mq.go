package mq

import "time"

// MQ 消息队列
type MQ interface {
	// Push 直接推送
	Push([]byte) error

	// Request 等待应答
	Request([]byte, time.Duration) ([]byte, error)
}
