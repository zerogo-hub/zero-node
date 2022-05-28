package kcp

// Config KCP 的一些专属配置
type Config struct {
	// streamMode 是否启用流模式
	streamMode bool
	// mtu 包 mtu，超过会拆包
	mtu int
	// sndwnd 发送窗口
	sndwnd int
	// rcvwnd 接收窗口
	rcvwnd int
	// datashard 凑齐多少包，开始生成冗余包
	datashard int
	// parityshard 冗余包生成个数
	parityshard int
	// nocomp 不压缩
	nocomp bool
	// acknodelay 延迟 ack
	acknodelay bool
	// nodelay 是否启用 nodelay模式，0不启用；1启用
	nodelay int
	// interval 内部轮询周期，毫秒，多久处理一批包
	interval int
	// resend 快速重传，N次ACK跨越将会直接重传，0 表示关闭
	resend int
	// nc 是否关闭流控，0 不关闭，1 关闭
	nc int
	// sockbuf socket 缓冲区
	sockbuf int
	// tcp 是否使用 tcp 传输
	tcp bool
}

func defaultConfig() *Config {
	return &Config{
		streamMode:  true,
		mtu:         1400,
		sndwnd:      1024,
		rcvwnd:      1024,
		datashard:   10,
		parityshard: 3,
		nocomp:      true,
		acknodelay:  false,
		nodelay:     1,
		interval:    40,
		resend:      2,
		nc:          1,
		sockbuf:     4096,
		tcp:         false,
	}
}

// Option 设置配置选项
type Option func(*server)

// WithStreamMode 是否启用流模式
func WithStreamMode(streamMode bool) Option {
	return func(s *server) {
		s.kcpConfig.streamMode = streamMode
	}
}

// WithMTU 包 mtu，超过会拆包
func WithMTU(mtu int) Option {
	return func(s *server) {
		s.kcpConfig.mtu = mtu
	}
}

// WithSndwnd 发送窗口
func WithSndwnd(sndwnd int) Option {
	return func(s *server) {
		s.kcpConfig.sndwnd = sndwnd
	}
}

// WithRcvwnd 接收窗口
func WithRcvwnd(rcvwnd int) Option {
	return func(s *server) {
		s.kcpConfig.rcvwnd = rcvwnd
	}
}

// WithDatashard 凑齐多少包，开始生成冗余包
func WithDatashard(datashard int) Option {
	return func(s *server) {
		s.kcpConfig.datashard = datashard
	}
}

// WithParityshard 冗余包生成个数
func WithParityshard(parityshard int) Option {
	return func(s *server) {
		s.kcpConfig.parityshard = parityshard
	}
}

// WithNocomp 不压缩
func WithNocomp(nocomp bool) Option {
	return func(s *server) {
		s.kcpConfig.nocomp = nocomp
	}
}

// WithAcknodelay 延迟 ack
func WithAcknodelay(acknodelay bool) Option {
	return func(s *server) {
		s.kcpConfig.acknodelay = acknodelay
	}
}

// WithNodelay 开启nodelay，RTO=RTO+0.5RTO，否则RTO=RTO+RTO
func WithNodelay(nodelay int) Option {
	return func(s *server) {
		s.kcpConfig.nodelay = nodelay
	}
}

// WithInterval 内部轮询周期，多久处理一批包
func WithInterval(interval int) Option {
	return func(s *server) {
		s.kcpConfig.interval = interval
	}
}

// WithResend 快速重传，N次ACK跨越将会直接重传
func WithResend(resend int) Option {
	return func(s *server) {
		s.kcpConfig.resend = resend
	}
}

// WithNC 不开拥塞控制，即不退让，只受收发窗口控制
func WithNC(nc int) Option {
	return func(s *server) {
		s.kcpConfig.nc = nc
	}
}

// WithSockbuf socket 缓冲区
func WithSockbuf(sockbuf int) Option {
	return func(s *server) {
		s.kcpConfig.sockbuf = sockbuf
	}
}

// WithTCP 是否使用 tcp 传输
func WithTCP(tcp bool) Option {
	return func(s *server) {
		s.kcpConfig.tcp = tcp
	}
}
