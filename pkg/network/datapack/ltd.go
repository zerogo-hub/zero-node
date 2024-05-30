package datapack

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"unsafe"

	zeroringbytes "github.com/zerogo-hub/zero-helper/buffer/ringbytes"
	zerobytes "github.com/zerogo-hub/zero-helper/bytes"
	zerocompress "github.com/zerogo-hub/zero-helper/compress"
	zerocrypto "github.com/zerogo-hub/zero-helper/crypto"
	zerologger "github.com/zerogo-hub/zero-helper/logger"
	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
)

var (
	// ErrGetPayloadLen 获取负载长度失败
	ErrGetPayloadLen = errors.New("get payload length failed")

	// ErrGetAllBytes 获取所有内容失败
	ErrGetAllBytes = errors.New("get all bytes failed")

	// ErrVerifyChecksum 校验失败
	ErrVerifyChecksum = errors.New("verify checksum failed")

	// ErrNoChecksumFlag 无校验标记
	ErrNoChecksumFlag = errors.New("no checksum flag")

	// ErrDecryptPayload 解密负载失败
	ErrDecryptPayload = errors.New("decrypt payload failed")

	// ErrDecompressPayload 解压负载失败
	ErrDecompressPayload = errors.New("decompress payload failed")
)

const (
	ChecksumLength = 16
)

// ltdMessageHead 消息头
type ltdMessageHead struct {
	// Len 包体长度，即 ltdMessageBody 的长度
	Len uint16
	// Flag 标记，具体见 modules/network/flag.go
	Flag uint16
	// SN 自增编号，由客户端发出，服务端原样返回。服务端主动发出的消息中 SN 值为 0
	SN uint16
	// Checksum 校验值
	Checksum [ChecksumLength]byte
}

// ltdMessageBody 消息体
type ltdMessageBody struct {
	// Code 错误码，如果存在错误，则会在 payload 中存储具体的错误信息
	Code uint16
	// Module 功能模块，用来表示一个功能大类，比如商店、副本
	Module uint8
	// Action 功能细分，用来表示一个功能里面的具体功能，比如进入副本，退出副本
	Action uint8
	// Payload 具体内容
	Payload []byte
}

// HeadLen 消息头长度，6 字节或者 22 字节
func ltdHeadLen(whetherChecksum bool) int {
	length := int(unsafe.Sizeof(ltdMessageHead{}))

	if !whetherChecksum {
		length -= ChecksumLength
	}

	return length
}

// ltdMessage 消息
type ltdMessage struct {
	// head 消息头
	head *ltdMessageHead

	// body 消息体
	body *ltdMessageBody

	// sessionID 会话 id
	sessionID zeronetwork.SessionID
}

// NewLTDMessage 创建一个消息
func NewLTDMessage(flag, sn, code uint16, module, action uint8, payload []byte) zeronetwork.Message {
	m := messagePool.Get().(*ltdMessage)

	m.head.Len = uint16(4 + len(payload))
	m.head.Flag = flag
	m.head.SN = sn

	m.body.Code = code
	m.body.Module = module
	m.body.Action = action
	m.body.Payload = payload

	return m
}

// SessionID 会话 ID，每一个连接都有一个唯一的会话 ID
func (m *ltdMessage) SessionID() zeronetwork.SessionID {
	return m.sessionID
}

// SetSessionID 设置 sessionID
func (m *ltdMessage) SetSessionID(sessionID zeronetwork.SessionID) {
	m.sessionID = sessionID
}

// Code 错误码
func (m *ltdMessage) Code() uint16 {
	return m.body.Code
}

// ModuleID 功能模块，用来表示一个功能大类，比如商店、副本
func (m *ltdMessage) ModuleID() uint8 {
	return m.body.Module
}

// ActionID 功能细分，用来表示一个功能里面的具体功能，比如进入副本，退出副本
func (m *ltdMessage) ActionID() uint8 {
	return m.body.Action
}

// Flag 标记
func (m *ltdMessage) Flag() uint16 {
	return m.head.Flag
}

// SN 自增编号
func (m *ltdMessage) SN() uint16 {
	return m.head.SN
}

// Payload 负载
func (m *ltdMessage) Payload() []byte {
	return m.body.Payload
}

// Checksum 校验值
func (m *ltdMessage) Checksum() [ChecksumLength]byte {
	return m.head.Checksum
}

// String 打印信息
func (m *ltdMessage) String() string {
	return fmt.Sprintf("sn: %d, module: %d, action: %d", m.head.SN, m.body.Module, m.body.Action)
}

// Release 释放资源

func (m *ltdMessage) Release() {
	messagePool.Put(m)
}

// ltd 按 Length-Type-Data 格式进行封包与解包
// 封装出的消息结构见 ltdMessage
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

	// whetherChecksum 是否启用校验值功能
	whetherChecksum bool

	// order 默认使用大端模式
	order binary.ByteOrder

	// logger 日志
	logger zerologger.Logger

	// emptyChecksum 空检验值，用于计算
	emptyChecksum [ChecksumLength]byte
}

// NewLTD 创建一个封包解包工具
// Length-Type-Data
func NewLTD(
	whetherCompress bool,
	compressThreshold int,
	compress zerocompress.Compress,
	whetherCrypto bool,
	whetherChecksum bool,
	logger zerologger.Logger,
) zeronetwork.Datapack {
	return &ltd{
		headLen:           ltdHeadLen(whetherChecksum),
		whetherCompress:   whetherCompress,
		compressThreshold: compressThreshold,
		compress:          compress,
		whetherCrypto:     whetherCrypto,
		whetherChecksum:   whetherChecksum,
		// 默认使用大端，zerobytes.ToUint16 也是大端模式
		order:         binary.BigEndian,
		logger:        logger,
		emptyChecksum: [ChecksumLength]byte{},
	}
}

// HeadLen 消息头长度
func (l *ltd) HeadLen() int {
	return l.headLen
}

// Pack 封包
func (l *ltd) Pack(message zeronetwork.Message, crypto zeronetwork.Crypto, checksumKey []byte) ([]byte, error) {
	body, flag, err := l.packBody(message, crypto)
	if err != nil {
		return nil, err
	}

	// 校验值
	if l.whetherChecksum {
		flag |= zeronetwork.FlagChecksum
	}

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)
	buffer.Reset()

	bodyLen := uint16(len(body))

	// 消息体长度
	if err := binary.Write(buffer, l.order, bodyLen); err != nil {
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

	// 校验值
	if l.whetherChecksum {
		if err := binary.Write(buffer, l.order, l.emptyChecksum); err != nil {
			return nil, err
		}
	}
	// 负载
	if len(body) > 0 {
		if err := binary.Write(buffer, l.order, body); err != nil {
			return nil, err
		}
	}

	allBytes := buffer.Bytes()

	// 计算校验值并填充
	if l.whetherChecksum && (flag&zeronetwork.FlagZero == 0) {
		calcChecksum := zerocrypto.HmacMd5ByteToByte(allBytes, checksumKey)
		checksumStartIndex := l.HeadLen() - ChecksumLength
		for i, v := range calcChecksum {
			allBytes[checksumStartIndex+i] = v
		}
	}

	return allBytes, nil
}

func (l *ltd) packBody(message zeronetwork.Message, crypto zeronetwork.Crypto) ([]byte, uint16, error) {
	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)
	buffer.Reset()

	// 错误码
	if err := binary.Write(buffer, l.order, message.Code()); err != nil {
		return nil, 0, err
	}
	// Module
	if err := binary.Write(buffer, l.order, message.ModuleID()); err != nil {
		return nil, 0, err
	}
	// Action
	if err := binary.Write(buffer, l.order, message.ActionID()); err != nil {
		return nil, 0, err
	}
	// 负载
	payload := message.Payload()
	if len(payload) > 0 {
		if err := binary.Write(buffer, l.order, payload); err != nil {
			return nil, 0, err
		}
	}

	var err error
	body := buffer.Bytes()
	flag := message.Flag()

	// 压缩
	if l.whetherCompress && l.compress != nil && len(body) >= l.compressThreshold {
		body, err = l.compress.Compress(body)
		if err != nil {
			l.logger.Errorf("compress failed, message: %s, err: %s", message.String(), err.Error())
			return nil, 0, err
		}

		flag |= zeronetwork.FlagCompress
	}

	// 加密
	if l.whetherCrypto && crypto != nil && (flag&zeronetwork.FlagZero == 0) {
		body, err = crypto.Encrypt(body)
		if err != nil {
			l.logger.Errorf("encrypt failed, message: %s, err: %s", message.String(), err.Error())
			return nil, 0, err
		}

		flag |= zeronetwork.FlagEncrypt
	}

	return body, flag, nil
}

// Unpack 解包
func (l *ltd) Unpack(buffer *zeroringbytes.RingBytes, crypto zeronetwork.Crypto, checksumKey []byte) ([]zeronetwork.Message, error) {
	messages := []zeronetwork.Message{}

	for {
		bufferLen := buffer.Len()

		if bufferLen < l.headLen {
			// 内容连消息头都无法存放完，目前这不是一个完整的消息
			break
		}

		// 取出消息体长度
		p, err := buffer.Peek(2)
		if err != nil {
			return nil, ErrGetPayloadLen
		}
		bodyLen := int(zerobytes.ToUint16(p))
		index := 2

		// 判断是否满足至少一个消息
		if bufferLen < l.headLen+bodyLen {
			// 当前内容长度 < 消息头长度 + 负载长度
			// 目前这不是一个完整的消息
			break
		}

		// 取出所有内容
		allBytes, err := buffer.Read(l.headLen + bodyLen)
		if err != nil {
			return nil, ErrGetAllBytes
		}

		// ---------------------- 消息头 ----------------------

		// flag 标记
		p = allBytes[index : index+2]
		flag := zerobytes.ToUint16(p)
		index += 2

		// sn 自增编号
		p = allBytes[index : index+2]
		sn := zerobytes.ToUint16(p)
		index += 2

		// checksum 校验值
		if l.whetherChecksum {
			// 发送端需要设置此标记
			if flag&zeronetwork.FlagChecksum == 0 {
				return nil, ErrNoChecksumFlag
			}

			if flag&zeronetwork.FlagZero == 0 {
				checksum := [ChecksumLength]byte{}
				p = allBytes[index : index+ChecksumLength]
				copy(checksum[:], p)
				if !l.verifyChecksum(checksum, allBytes, checksumKey) {
					return nil, ErrVerifyChecksum
				}
			}

			index += ChecksumLength
		}

		// ---------------------- 消息体(解密、解压) ----------------------

		bodyBytes := allBytes[index:]

		// 解密
		if flag&zeronetwork.FlagEncrypt != 0 && crypto != nil && (flag&zeronetwork.FlagZero == 0) {
			bodyBytes, err = crypto.Decrypt(bodyBytes)
			if err != nil {
				l.logger.Errorf("decrypt failed, sn: %d, err: %s", sn, err.Error())
				return nil, ErrDecryptPayload
			}
		}

		// 解压
		if flag&zeronetwork.FlagCompress != 0 && l.compress != nil {
			bodyBytes, err = l.compress.Uncompress(bodyBytes)
			if err != nil {
				l.logger.Errorf("decompress failed, sn: %d, err: %s", sn, err.Error())
				return nil, ErrDecompressPayload
			}
		}

		index = 0

		// code 错误码
		code := uint16(0)
		index += 2

		// module 功能模块
		p = bodyBytes[index : index+1]
		module := zerobytes.ToUint8(p)
		index += 1

		// action 功能细分
		p = bodyBytes[index : index+1]
		action := zerobytes.ToUint8(p)
		index += 1

		// payload 负载
		var payload []byte
		if bodyLen-4 > 0 {
			payload = bodyBytes[index:]
		}

		// 组装一个消息
		message := NewLTDMessage(flag, sn, code, module, action, payload)
		messages = append(messages, message)
	}

	return messages, nil
}

func (l *ltd) verifyChecksum(checksum [ChecksumLength]byte, allBytes, checksumKey []byte) bool {
	// 将填写检验值部分置 0
	checksumStartIndex := l.HeadLen() - ChecksumLength
	for i := checksumStartIndex; i < checksumStartIndex+ChecksumLength; i++ {
		allBytes[i] = 0
	}
	calcChecksum := zerocrypto.HmacMd5ByteToByte(allBytes, checksumKey)

	if len(calcChecksum) != len(checksum) {
		return false
	}

	for i, v1 := range checksum {
		if v1 != calcChecksum[i] {
			return false
		}
	}

	return true
}

var bufferPool *sync.Pool
var messagePool *sync.Pool

func init() {
	bufferPool = &sync.Pool{}
	bufferPool.New = func() interface{} {
		return &bytes.Buffer{}
	}

	messagePool = &sync.Pool{}
	messagePool.New = func() interface{} {
		return &ltdMessage{
			head: &ltdMessageHead{},
			body: &ltdMessageBody{},
		}
	}
}
