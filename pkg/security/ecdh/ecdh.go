package ecdh

import (
	"bytes"
	"math/rand"
	"sync"

	libCurve "golang.org/x/crypto/curve25519"
)

type ExchangeRequest struct {
	// PublicKey 客户端公钥
	PublicKey string `json:"public_key"`

	// R 客户端随机数
	R string `json:"r"`
}

type ExchageResponse struct {
	// PublicKey 服务器公钥
	PublicKey string `json:"public_key"`

	// R 服务器随机数
	R string `json:"r"`
}

// GenerateKeys 生成公钥和私钥
func GenerateKeys() ([]byte, []byte) {
	var privateKey [32]byte
	for i := range privateKey[:] {
		privateKey[i] = byte(rand.Intn(256))
	}
	var publicKey [32]byte
	libCurve.ScalarBaseMult(&publicKey, &privateKey)

	return publicKey[:], privateKey[:]
}

// GenerateShareKey 使用私钥和对方的公钥生成共享秘钥
func GenerateShareKey(privateKey, targetPublicKey []byte) ([]byte, error) {
	sharedKey, err := libCurve.X25519(privateKey, targetPublicKey)
	return sharedKey, err
}

func BuildKey(sharedKey, rs, rc []byte) []byte {
	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)
	buffer.Reset()

	buffer.Write(sharedKey)
	buffer.Write(rs)
	buffer.Write(rc)

	return buffer.Bytes()
}

var bufferPool *sync.Pool

func init() {
	bufferPool = &sync.Pool{}
	bufferPool.New = func() interface{} {
		return &bytes.Buffer{}
	}
}
