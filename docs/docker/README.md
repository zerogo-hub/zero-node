## 网络分配

网络定义在`docs/docker/etcd`中，其余模块均使用这个网络

| 名称       | 分组  | IP           | Port       |
| ---------- | ----- | ------------ | ---------- |
| zero.etcd1 | etcd  | 172.19.10.11 | 8081, 8082 |
| zero.etcd2 | etcd  | 172.19.10.12 | 8083, 8084 |
| zero.etcd3 | etcd  | 172.19.10.13 | 8085, 8086 |
| zero.nats1 | nats  | 172.19.10.21 | 8091, 8092 |
| zero.nats2 | nats  | 172.19.10.22 | 8093, 8094 |
| zero.nats3 | nats  | 172.19.10.23 | 8095, 8096 |
| zero.redis | redis | 172.19.10.31 | 8087       |
