 .PHONY: mod

mod:
	go env
	go mod tidy
	go mod download
	go mod verify
	go mod vendor