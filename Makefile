default: build

build: *.go
	go build

.PHONY: run fmt

run: *.go
	go run main.go -pattern=axwwo

fmt: *.go
	go fmt *.go
