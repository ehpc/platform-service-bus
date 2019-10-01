all: format test build

format:
	go fmt ./...

test:
	go test ./...

build:
	go build -o platform-service-bus main.go
