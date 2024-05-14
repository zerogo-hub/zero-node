package network

import (
	"io"
	"net"
)

// IsEOFOrReadError 是否是连接结束或者是读取错误
// EOF
// closed by remote
func IsEOFOrReadError(err error) bool {
	if err == nil {
		return false
	}

	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return true
	}

	if e, ok := err.(*net.OpError); ok && e.Op == "read" {
		return true
	}

	return false
}
