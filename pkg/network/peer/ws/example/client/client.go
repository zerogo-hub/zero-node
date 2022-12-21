package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	protocol "github.com/zerogo-hub/zero-node/pkg/network/peer/tcp/example/protocol"

	zerocodec "github.com/zerogo-hub/zero-helper/codec"
	zeroprotobuf "github.com/zerogo-hub/zero-helper/codec/protobuf"
	zerozip "github.com/zerogo-hub/zero-helper/compress/zip"

	zeronetwork "github.com/zerogo-hub/zero-node/pkg/network"
	zerodatapack "github.com/zerogo-hub/zero-node/pkg/network/datapack"
	zerows "github.com/zerogo-hub/zero-node/pkg/network/peer/ws"
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

type client struct {
	cc zeronetwork.Client

	sn uint16

	// router 路由
	router zeronetwork.Router

	// codec 编码与解码器
	codec zerocodec.Codec

	// closeCh 停止信号
	closeCh chan bool
}

func main() {
	c := &client{
		router:  zeronetwork.NewRouter(),
		codec:   zeroprotobuf.NewProtobufCodec(),
		closeCh: make(chan bool),
	}

	// 注册路由
	_ = c.router.AddRouter(ModuleHello, ActionHelloSayResp, c.respSayHello)

	// 测试用例的 ssl 证书是自签名的，此处忽略验证
	insecureSkipVerify := true

	// 创建客户端，添加路由处理服务端的响应
	cc := zerows.NewClient(
		websocket.BinaryMessage,
		insecureSkipVerify,
		c.router.Handler,

		// 要对消息进行压缩和解压
		zerows.WithClientWhetherCompress(true),
		// 指定压缩和解压的方式
		zerows.WithClientCompress(zerozip.NewZip()),
		// 指定压缩的阈值，负载长度超过此值才会进行压缩
		zerows.WithClientCompressThreshold(2),

		// 要对消息进行加密
		zerows.WithClientWhetherCrypto(true),
	)
	if err := cc.Connect("wss", "localhost", 8001); err != nil {
		cc.Logger().Errorf("connect failed, err: %s", err.Error())
		return
	}

	// 设置加密与解密的工具
	crypto, _ := zerorc4.New(secretKey)
	cc.SetCrypto(crypto)

	c.cc = cc
	c.start()
}

func (c *client) start() {
	go c.cc.Run()

	// 主动发起消息
	go c.ping()

	c.signal()
}

func (c *client) ping() {
	for {
		select {
		case <-time.After(1000 * time.Millisecond):
			if err := c.reqSayHello(); err != nil {
				c.cc.Logger().Errorf("sayHelloReq failed: %s", err.Error())
				return
			}
		case <-c.closeCh:
			return
		}
	}
}

// signal 监听信号
func (c *client) signal() {
	// ctrl + c 或者 kill
	sigs := []os.Signal{syscall.SIGINT, syscall.SIGTERM}

	ch := make(chan os.Signal, 1)

	signal.Notify(ch, sigs...)

	sig := <-ch

	signal.Stop(ch)

	c.closeCh <- true

	c.cc.Logger().Infof("received signal, sig: %+v, exit now", sig)
}

func (c *client) reqSayHello() error {
	req, _ := c.codec.Marshal(&protocol.Req1{
		Name: "MyClient",
		Word: "Hello MyServer",
	})

	return c.send(ModuleHello, ActionHelloSayReq, req)
}

func (c *client) respSayHello(message zeronetwork.Message) (zeronetwork.Message, error) {
	res := &protocol.Resp1{}
	if err := c.codec.Unmarshal(message.Payload(), res); err != nil {
		return nil, err
	}

	c.cc.Logger().Infof("recv from server: %s", res.Word)
	return nil, nil
}

func (c *client) send(module, action uint8, payload []byte) error {
	flag := uint16(0)
	c.sn++
	code := uint16(0)
	message := zerodatapack.NewLTDMessage(flag, c.sn, code, module, action, payload)
	return c.cc.Send(message)
}
