package datapack

import (
	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
)

// DefaultDatapck 默认的封包与解包器
func DefaultDatapck(config *zeronetwork.Config) zeronetwork.Datapack {
	return NewLTD(
		config.WhetherCompress,
		config.CompressThreshold,
		config.Compress,
		config.WhetherCrypto,
		config.Logger,
	)
}
