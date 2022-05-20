# zero-node

用于组装一个游戏节点

# 模块

- network: 网络模块，支持`tcp`和`websocket`，使用[https://github.com/gorilla/websocket](gorilla/websocket)实现`websocket`
- rpc: 远程调用，基于[https://doc.rpcx.io/](rpcx)实现
- event: 事件派发，分为本进程派发和跨进程派发
- timer: 定时器

# 第三方模块

- ants: 线程池
- gorilla/websocket: Websocket 框架
- rpcx: RPC 框架
