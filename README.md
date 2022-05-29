# zero-node

用于构建分布式服务器，具体使用见[zero-server](https://github.com/zerogo-hub/zero-server)

# 模块

- actor: `Actor`模式
- ecs: `ECS`支持
- event: 事件派发，分为本进程派发和跨进程派发
- entity: 封装数据库与缓存的访问，要求`go 1.8` 以上
- network: 网络模块，支持`tcp`、`kcp`和`websocket`
  - 使用[kcp-go](https://github.com/xtaci/kcp-go)实现`kcp`
  - 使用[gorilla/websocket](https://github.com/gorilla/websocket)实现`websocket`
- rpc: 远程调用
  - 使用[rpcx](<(https://doc.rpcx.io)>)实现
- security: 安全相关
  - dh: 密钥交换算法
  - rc4: 对称加密
  - srp6: 远程安全登录
- timer: 定时器

# 第三方模块

- ants: 线程池
- gorilla/websocket: Websocket 框架
- protobuf: Google protobuf
- rpcx: RPC 框架

# 安装必要模块

- protobuf

  ```text
  brew install protobuf
  brew install protoc-gen-go
  ```
