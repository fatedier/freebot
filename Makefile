all: fmt build

build: freebot

fmt:
	go fmt ./...

freebot:
	go build -o ./bin/freebot ./cmd
