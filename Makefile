default: run
build: *.go
	go build
run: *.go
	go run main.go -pattern=axwwo
fmt: *.go
	go fmt *.go
