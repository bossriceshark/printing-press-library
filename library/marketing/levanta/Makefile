.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -trimpath -o bin/levanta-pp-cli$(BIN_EXT) ./cmd/levanta-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/levanta-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -trimpath -o bin/levanta-pp-mcp$(BIN_EXT) ./cmd/levanta-pp-mcp

install-mcp:
	go install ./cmd/levanta-pp-mcp

build-all: build build-mcp
