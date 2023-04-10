package datapack

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sync"

	zerocircle "github.com/zerogo-hub/zero-helper/buffer/circle"
	zerobytes "github.com/zerogo-hub/zero-helper/bytes"
	zerocompress "github.com/zerogo-hub/zero-helper/compress"
	zerologger "github.com/zerogo-hub/zero-helper/logger"
	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
)

var (
	// ErrGetPayloadLen 获取负载长度失败
	ErrGetPayloadLen = errors.New("get payload length failed")

	// ErrSkip buff 跳过失败
	ErrSkip = errors.New("buffer skip failed")

	// ErrGetFlag 获取标记失败
	ErrGetFlag = errors.New("get flag failed")

	// ErrGetSN 获取自增编号失败
	ErrGetSN = errors.New("get sn failed")

	// ErrGetModule 获取功能模块信息失败
	ErrGetModule = errors.New("get module id failed")

	// ErrGetAction 获取功能细分信息失败
	ErrGetAction = errors.New("get action id failed")

	// ErrGetPayload 获取负载失败
	ErrGetPayload = errors.New("get payload failed")

	// ErrDecryptPayload 解密负载失败
	ErrDecryptPayload = errors.New("decrypt payload failed")

	// ErrDecompressPayload 解压负载失败
	ErrDecompressPayload = errors.New("decompress payload failed")
)

// ltd 按 Length-Type-Data 格式进行封包与解包
// 封装出的消息结构见 ltd-message.go/ltdMessage
type ltd struct {
	// headLen 消息头长度
	headLen int

	// whetherCompress 是否需要对消息负载 payload 进行压缩
	whetherCompress bool

	// compressThreshold 压缩的阈值，当消息负载 payload 长度不小于该值时才会压缩
	compressThreshold int

	// compress 压缩与解压器，默认 zip
	compress zerocompress.Compress

	// whetherCrypto 是否需要对消息负载 payload 进行加密
	whetherCrypto bool

	// order 默认使用大端模式
	order binary.ByteOrder

	// logger 日志
	logger zerologger.Logger
}

// NewLTD 创建一个封包解包工具
func NewLTD(
	whetherCompress bool,
	compressThreshold int,
	compress zerocompress.Compress,
	whetherCrypto bool,
	logger zerologger.Logger,
) zeronetwork.Datapack {
	return &ltd{
		headLen:           ltdHeadLen(),
		whetherCompress:   whetherCompress,
		compressThreshold: compressThreshold,
		compress:          compress,
		whetherCrypto:     whetherCrypto,
		// 默认使用大端，zerobytes.ToUint16 也是大端模式
		order:  binary.BigEndian,
		logger: logger,
	}
}

// HeadLen 消息头长度
func (l *ltd) HeadLen() int {
	return l.headLen
}

// Pack 封包
func (l *ltd) Pack(message zeronetwork.Message, crypto zeronetwork.Crypto) ([]byte, error) {
	var err error

	// 处理负载：压缩，加密
	flag := message.Flag()
	payload := message.Payload()

	if len(payload) > 0 {
		// 压缩
		if l.whetherCompress && len(payload) >= l.compressThreshold && l.compress != nil {
			payload, err = l.compress.Compress(payload)
			if err != nil {
				l.logger.Errorf("compress failed, message: %s, err: %s", message.String(), err.Error())
				return nil, err
			}

			flag |= zeronetwork.FlagCompress
		}

		// 加密
		if l.whetherCrypto && crypto != nil {
			payload, err = crypto.Encrypt(payload)
			if err != nil {
				l.logger.Errorf("encrypt failed, message: %s, err: %s", message.String(), err.Error())
				return nil, err
			}

			flag |= zeronetwork.FlagEncrypt
		}
	}

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)
	buffer.Reset()

	payloadLen := uint16(len(payload))

	// 负载长度
	if err := binary.Write(buffer, l.order, payloadLen); err != nil {
		return nil, err
	}
	// flag 标记
	if err := binary.Write(buffer, l.order, flag); err != nil {
		return nil, err
	}
	// SN 编号
	if err := binary.Write(buffer, l.order, message.SN()); err != nil {
		return nil, err
	}
	// 错误码
	if err := binary.Write(buffer, l.order, message.Code()); err != nil {
		return nil, err
	}
	// Module
	if err := binary.Write(buffer, l.order, message.ModuleID()); err != nil {
		return nil, err
	}
	// Action
	if err := binary.Write(buffer, l.order, message.ActionID()); err != nil {
		return nil, err
	}
	// 负载
	if len(payload) > 0 {
		if err := binary.Write(buffer, l.order, payload); err != nil {
			return nil, err
		}
	}

	return buffer.Bytes(), nil
}

// Unpack 解包
func (l *ltd) Unpack(buffer *zerocircle.Circle, crypto zeronetwork.Crypto) ([]zeronetwork.Message, error) {
	messages := []zeronetwork.Message{}

	for {
		bufferLen := buffer.Len()

		if bufferLen < l.headLen {
			// 内容连消息头都无法存放完，目前这不是一个完整的消息
			break
		}

		// 取出负载长度
		p, err := buffer.Peek(2)
		if err != nil {
			return nil, ErrGetPayloadLen
		}
		payloadLen := int(zerobytes.ToUint16(p))

		// 判断是否满足至少一个消息
		if bufferLen < l.headLen+payloadLen {
			// 当前内容长度 < 消息头长度 + 负载长度
			// 目前这不是一个完整的消息
			break
		}

		// 至少有一个完整的消息
		if err := buffer.Skip(2); err != nil {
			return nil, ErrSkip
		}

		// flag 标记
		p, err = buffer.Get(2)
		if err != nil {
			return nil, ErrGetFlag
		}
		flag := zerobytes.ToUint16(p)

		// sn 自增编号
		p, err = buffer.Get(2)
		if err != nil {
			return nil, ErrGetSN
		}
		sn := zerobytes.ToUint16(p)

		// code 错误码
		if err := buffer.Skip(2); err != nil {
			return nil, ErrSkip
		}
		code := uint16(0)

		// module 功能模块
		p, err = buffer.Get(1)
		if err != nil {
			return nil, ErrGetModule
		}
		module := zerobytes.ToUint8(p)

		// action 功能细分
		p, err = buffer.Get(1)
		if err != nil {
			return nil, ErrGetModule
		}
		action := zerobytes.ToUint8(p)

		// payload 负载
		var payload []byte
		if payloadLen > 0 {
			payload, err = buffer.Get(payloadLen)
			if err != nil {
				return nil, ErrGetPayload
			}
		}

		if len(payload) > 0 {
			// 解密
			if flag&zeronetwork.FlagEncrypt != 0 && crypto != nil {
				payload, err = crypto.Decrypt(payload)
				if err != nil {
					l.logger.Errorf("decrypt failed, module: %d, action: %d, err: %s", module, action, err.Error())
					return nil, ErrDecryptPayload
				}
			}

			// 解压
			if flag&zeronetwork.FlagCompress != 0 && l.compress != nil {
				payload, err = l.compress.Uncompress(payload)
				if err != nil {
					l.logger.Errorf("decompress failed, module: %d, action: %d, err: %s", module, action, err.Error())
					return nil, ErrDecompressPayload
				}
			}
		}

		// 组装一个消息
		message := NewLTDMessage(flag, sn, code, module, action, payload)
		messages = append(messages, message)
	}

	return messages, nil
}

var bufferPool *sync.Pool

func init() {
	bufferPool = &sync.Pool{}
	bufferPool.New = func() interface{} {
		return &bytes.Buffer{}
	}
}
