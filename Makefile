all: format test build

format:
	go fmt ./...

test:
	go test ./...

build:
	go build -o platfrom-service-bus main.go
