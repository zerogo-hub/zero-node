package rc4

import (
	stdRC4 "crypto/rc4"

	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
)

type rc4 struct {
	// 加密和解密不能使用同一个 Cipher 对象
	cipherEn *stdRC4.Cipher
	cipherDe *stdRC4.Cipher
}

// New 加密和解密要分别创建实例
func New(key []byte) (zeronetwork.Crypto, error) {
	cipherEn, err := stdRC4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	cipherDe, err := stdRC4.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return &rc4{cipherEn: cipherEn, cipherDe: cipherDe}, nil
}

// Encrypt 加密
func (rc4 *rc4) Encrypt(in []byte) ([]byte, error) {
	out := make([]byte, len(in))
	rc4.cipherEn.XORKeyStream(out, in)
	return out, nil
}

// Decrypt 解密
func (rc4 *rc4) Decrypt(in []byte) ([]byte, error) {
	out := make([]byte, len(in))
	rc4.cipherDe.XORKeyStream(out, in)
	return out, nil
}
