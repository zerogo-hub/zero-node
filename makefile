 .PHONY: mod

mod:
	go env
	go mod tidy
	go mod download
	go mod verify
	go mod vendor

lint:
	golangci-lint run

# 创建自签名的 ssl 证书
# 输入 common name 时，输入对应的域名，或者输入 127.0.0.1
# 创建好的证书: network/peer/ws/example/server
ssl:
	openssl genrsa -aes256 -passout pass:123456 -out server.pass.key 4096
	openssl rsa -passin pass:123456 -in server.pass.key -out server.key
	rm server.pass.key
	openssl req -new -key server.key -out server.csr
	openssl x509 -req -sha256 -days 3650 -in server.csr -signkey server.key -out server.crt
