package ecdh_test

import (
	"encoding/hex"
	"reflect"
	"testing"

	zerorandom "github.com/zerogo-hub/zero-helper/random"
	zeroecdh "github.com/zerogo-hub/zero-node/pkg/security/ecdh"
)

func TestExchangeKey(t *testing.T) {
	// 客户端 --------------------------------------------
	clientPublicKey, clientPrivateKey := zeroecdh.GenerateKeys()
	clientRandomValue := zerorandom.Bytes(32)

	request := &zeroecdh.ExchangeRequest{
		PublicKey: hex.EncodeToString(clientPublicKey),
		R:         hex.EncodeToString(clientRandomValue),
	}

	// 服务端 --------------------------------------------
	// 解析出客户端请求的公钥和随机值
	peerClientPublicKey, _ := hex.DecodeString(request.PublicKey)
	peerClientRandomValue, _ := hex.DecodeString(request.R)

	serverPublicKey, serverPrivateKey := zeroecdh.GenerateKeys()
	serverRandomValue := zerorandom.Bytes(32)

	// 生成共享秘钥
	serverSharedKey, _ := zeroecdh.GenerateShareKey(serverPrivateKey, peerClientPublicKey)

	// 生成最终需要的秘钥
	serverKey := zeroecdh.BuildKey(serverSharedKey, serverRandomValue, peerClientRandomValue)

	response := &zeroecdh.ExchageResponse{
		PublicKey: hex.EncodeToString(serverPublicKey),
		R:         hex.EncodeToString(serverRandomValue),
	}

	// 客户端 --------------------------------------------
	// 解析出服务端返回的公钥和随机值
	peerServerPublicKey, _ := hex.DecodeString(response.PublicKey)
	peerServerRandomValue, _ := hex.DecodeString(response.R)

	// 生成共享秘钥
	clientSharedKey, _ := zeroecdh.GenerateShareKey(clientPrivateKey, peerServerPublicKey)

	// 生成最终需要的秘钥
	clientKey := zeroecdh.BuildKey(clientSharedKey, peerServerRandomValue, clientRandomValue)

	// 验证 --------------------------------------------
	if !reflect.DeepEqual(serverKey, clientKey) {
		t.Errorf("Unexpected key, serverKey: %#v, clientKey: %#v", serverKey, clientKey)
	}
}
