package nats

import (
	"time"

	"github.com/nats-io/nats.go"

	zeromq "github.com/zerogo-hub/zero-node/pkg/mq"
)

type proxy struct {
	conn *nats.Conn
}

// New ..
func New() zeromq.MQ {
	return &proxy{}
}

// Push 直接推送到目标
func (p *proxy) Push(payload []byte) error {
	return nil
}

// Request 等待应答
func (p *proxy) Request(payload []byte, tiemout time.Duration) ([]byte, error) {
	return nil, nil
}
