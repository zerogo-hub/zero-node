package main

import (
	"net/http"
	_ "net/http/pprof"

	protocol "github.com/zerogo-hub/zero-node/pkg/network/peer/tcp/example/protocol"

	zerocodec "github.com/zerogo-hub/zero-helper/codec"
	zeroprotobuf "github.com/zerogo-hub/zero-helper/codec/protobuf"
	zerozip "github.com/zerogo-hub/zero-helper/compress/zip"

	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
	zerodatapack "github.com/zerogo-hub/zero-node/pkg/network/datapack"
	zerotcp "github.com/zerogo-hub/zero-node/pkg/network/peer/tcp"
	zerorc4 "github.com/zerogo-hub/zero-node/pkg/security/rc4"
)

const (
	// ModuleHello hello 模块
	ModuleHello = 1

	// ActionHelloSayReq hello 模块 客户端请求
	ActionHelloSayReq = 1

	// ActionHelloSayResp hello 模块 服务端响应
	ActionHelloSayResp = 2
)

const (
	secretKey = "PUmjGmE9xccKlDWV"
)

type server struct {
	p zeronetwork.Peer

	// codec 编码与解码器
	codec zerocodec.Codec
}

func main() {
	s := &server{
		codec: zeroprotobuf.NewProtobufCodec(),
	}

	s.p = zerotcp.NewServer().WithOption(
		// 当服务器刚启动时
		zeronetwork.WithOnServerStart(s.onServerStart),
		// 当服务器已关闭后
		zeronetwork.WithOnServerClose(s.onServerClose),

		// 当有连接到来时
		zeronetwork.WithOnConnected(s.onConnected),
		// 当有连接关闭时
		zeronetwork.WithOnConnClose(s.onConnClose),

		// 要对消息进行压缩和解压
		zeronetwork.WithWhetherCompress(true),
		// 指定压缩和解压的方式
		zeronetwork.WithCompress(zerozip.NewZip()),
		// 指定压缩的阈值，负载长度超过此值才会进行压缩
		zeronetwork.WithCompressThreshold(64),

		// 要对消息进行加密
		zeronetwork.WithWhetherCrypto(true),
	)

	// pprof
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	s.p.Logger().Info("pprof: http://localhost:6060/debug/pprof/")

	// 注册路由
	s.p.Router().AddRouter(ModuleHello, ActionHelloSayReq, s.reqSayHello)

	s.p.Start()

	s.p.ListenSignal()
}

func (s *server) onServerStart() error {
	// 服务器启动时调用，可以添加一些初始化操作
	s.p.Logger().Info("server start, init success")
	return nil
}

func (s *server) onServerClose() {
	// 服务器启动时调用，可以添加一些初始化操作
	s.p.Logger().Info("server closed")
}

func (s *server) onConnected(session zeronetwork.Session) {
	s.p.Logger().Infof("session: %d connected, total: %d", session.ID(), s.p.SessionManager().Len())

	// 通过 dh 协议双方交换密钥用于后续加密
	// 这里直接使用 secretKey
	crypto, _ := zerorc4.New(secretKey)
	session.SetCrypto(crypto)
}

func (s *server) onConnClose(session zeronetwork.Session) {
	s.p.Logger().Infof("session: %d closed, remain: %d", session.ID(), s.p.SessionManager().Len())
}

func (s *server) reqSayHello(message zeronetwork.Message) (zeronetwork.Message, error) {
	// 客户端请求
	req := &protocol.Req1{}
	if err := s.codec.Unmarshal(message.Payload(), req); err != nil {
		return nil, err
	}
	s.p.Logger().Infof("recv from client: %d, message: %s, name: %s, word: %s", message.SessionID(), message.String(), req.Name, req.Word)

	// 响应
	res, err := s.codec.Marshal(&protocol.Resp1{
		Word: "respone from server",
	})
	if err != nil {
		return nil, err
	}

	return s.newMessage(message.SN(), ModuleHello, ActionHelloSayResp, res), nil
}

func (s *server) newMessage(sn uint16, module, action uint8, payload []byte) zeronetwork.Message {
	flag := uint16(0)
	code := uint16(0)
	message := zerodatapack.NewLTDMessage(flag, sn, code, module, action, payload)
	return message
}
