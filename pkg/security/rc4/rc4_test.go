package rc4_test

import (
	"testing"

	zerorc4 "github.com/zerogo-hub/zero-node/pkg/security/rc4"
)

func TestRC4(t *testing.T) {
	words := []byte("1234abcd")

	crypto, _ := zerorc4.New("12345678")
	encrypted, err := crypto.Encrypt(words)
	if err != nil {
		t.Fatal("encrypt failed")
	}

	decrypted, err := crypto.Decrypt(encrypted)
	if err != nil {
		t.Fatal("decrypt failed")
	}

	t.Log(decrypted)
}
