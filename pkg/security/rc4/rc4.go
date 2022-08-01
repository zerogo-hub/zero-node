package rc4

import (
	stdRC4 "crypto/rc4"

	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
)

type rc4 struct {
	cipher *stdRC4.Cipher
}

// New 加密和解密要分开使用
func New(key string) (zeronetwork.Crypto, error) {
	cipher, err := stdRC4.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	return &rc4{cipher: cipher}, nil
}

// Encrypt 加密
func (rc4 *rc4) Encrypt(in []byte) ([]byte, error) {
	out := make([]byte, len(in))
	rc4.cipher.XORKeyStream(out, in)
	return out, nil
}

// Decrypt 解密
func (rc4 *rc4) Decrypt(in []byte) ([]byte, error) {
	out := make([]byte, len(in))
	rc4.cipher.XORKeyStream(out, in)
	return out, nil
}
