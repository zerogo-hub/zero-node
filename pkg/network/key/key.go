package key

import (
	"encoding/hex"
	"errors"

	zerojson "github.com/zerogo-hub/zero-helper/json"
	zerorandom "github.com/zerogo-hub/zero-helper/random"
	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
	zerodatapack "github.com/zerogo-hub/zero-node/pkg/network/datapack"
	zeroecdh "github.com/zerogo-hub/zero-node/pkg/security/ecdh"
)

// ExchangeKeyRequest 创建秘钥协商，请求
// return: 私钥，随机值，请求消息
func ExchangeKeyRequest() ([]byte, []byte, zeronetwork.Message) {
	// 1. 生成公钥，私钥，随机数
	publicKey, privateKey := zeroecdh.GenerateKeys()
	randomValue := zerorandom.Bytes(32)

	// 2. 创建协商协议
	request := &zeroecdh.ExchangeRequest{
		PublicKey: hex.EncodeToString(publicKey),
		R:         hex.EncodeToString(randomValue),
	}
	payload, _ := zerojson.Marshal(request)

	flag := zeronetwork.FlagZero
	sn := uint16(0)
	code := uint16(0)
	module := uint8(0)
	action := zeronetwork.FlagZeroExchangeKeyRequest
	message := zerodatapack.NewLTDMessage(flag, sn, code, module, action, payload)

	return privateKey, randomValue, message
}

// ExchangeKeyResponse 响应秘钥协商
// return: 服务端最终秘钥，响应消息，错误
func ExchangeKeyResponse(requestBytes []byte) ([]byte, zeronetwork.Message, error) {
	// 1. 解析请求
	if len(requestBytes) == 0 {
		return nil, nil, errors.New("requestBytes is empty")
	}
	var request zeroecdh.ExchangeRequest
	if err := zerojson.Unmarshal(requestBytes, &request); err != nil {
		return nil, nil, err
	}
	peerClientPublicKey, _ := hex.DecodeString(request.PublicKey)
	peerClientRandomValue, _ := hex.DecodeString(request.R)

	// 2. 生成公钥，私钥，随机数
	publicKey, privateKey := zeroecdh.GenerateKeys()
	randomValue := zerorandom.Bytes(32)

	// 3. 生成共享秘钥
	serverSharedKey, _ := zeroecdh.GenerateShareKey(privateKey, peerClientPublicKey)

	// 4. 生成最终需要的秘钥
	key := zeroecdh.BuildKey(serverSharedKey, randomValue, peerClientRandomValue)

	// 5. 发送协商协议
	response := &zeroecdh.ExchageResponse{
		PublicKey: hex.EncodeToString(publicKey),
		R:         hex.EncodeToString(randomValue),
	}

	payload, _ := zerojson.Marshal(response)

	flag := zeronetwork.FlagZero
	sn := uint16(0)
	code := uint16(0)
	module := uint8(0)
	action := zeronetwork.FlagZeroExchangeKeyResponse
	message := zerodatapack.NewLTDMessage(flag, sn, code, module, action, payload)

	return key, message, nil
}

// ExchangeKeyParseResponse 解析秘钥协商的响应
// return 客户端最终秘钥，错误
func ExchangeKeyParseResponse(responseBytes, privateKey, randomValue []byte) ([]byte, error) {
	// 1. 解析响应
	if len(responseBytes) == 0 {
		return nil, errors.New("responseBytes is empty")
	}
	var response zeroecdh.ExchageResponse
	if err := zerojson.Unmarshal(responseBytes, &response); err != nil {
		return nil, err
	}
	peerServerPublicKey, _ := hex.DecodeString(response.PublicKey)
	peerServerRandomValue, _ := hex.DecodeString(response.R)

	// 2. 生成共享秘钥
	clientSharedKey, _ := zeroecdh.GenerateShareKey(privateKey, peerServerPublicKey)

	// 3. 生成最终需要的秘钥
	key := zeroecdh.BuildKey(clientSharedKey, peerServerRandomValue, randomValue)

	return key, nil
}
