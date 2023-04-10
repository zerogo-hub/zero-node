# zero-node

用于构建分布式服务器，具体使用见[zero-server](https://github.com/zerogo-hub/zero-server)

# 模块

- actor

- cluster

  - gate: 封装网关模块
  - node: 封装内部节点
  - robot: 机器人节点
  - web: http 节点

- component: 模块化

- config: 游戏使用的配置文件

  - local: 从本地本地 excel 文件，并监控文件变化
  - remote: 从专门的接口获取 json 文件

- mq: 消息队列

  - nats

- network: 网络模块，支持`tcp`、`kcp`和`websocket`

  - 使用[kcp-go](https://github.com/xtaci/kcp-go)实现`kcp`
  - 使用[gorilla/websocket](https://github.com/gorilla/websocket)实现`websocket`

- rpc: 封装 `rpcx-go`

- security: 安全相关

  - dh: 密钥交换算法
  - rc4: 对称加密
  - srp6: 远程安全登录

# 第三方模块

- gorilla/websocket
- protobuf
- nats-go

# 安装必要模块

- protobuf

  ```text
  brew install protobuf
  brew install protoc-gen-go
  ```

# TODO

[ ] 限流

# 游戏示例

- [ ] 服务端 [zero-game-server](https://github.com/zerogo-hub/zero-game-server)
- [ ] 客户端 [zero-game-client](https://github.com/zerogo-hub/zero-game-client)
- [ ] unity 客户端 [zero-game-client-unity](https://github.com/zerogo-hub/zero-game-client-unity)
- [ ] cocos creator 客户端 [zero-game-client-cocos](https://github.com/zerogo-hub/zero-game-client-cocos)
