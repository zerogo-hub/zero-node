.PHONY: build

build:
	protoc --go_out=./protocol *.proto

pprof:
	go tool pprof -http=":6061" ./server/server.go  http://localhost:6060/debug/pprof/heap
