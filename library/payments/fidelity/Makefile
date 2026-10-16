.PHONY: build test lint install clean

BIN_EXT := $(if $(filter windows,$(shell go env GOOS)),.exe,)

build:
	go build -trimpath -o bin/fidelity-pp-cli$(BIN_EXT) ./cmd/fidelity-pp-cli

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install ./cmd/fidelity-pp-cli

clean:
	rm -rf bin/

build-mcp:
	go build -trimpath -o bin/fidelity-pp-mcp$(BIN_EXT) ./cmd/fidelity-pp-mcp

install-mcp:
	go install ./cmd/fidelity-pp-mcp

build-all: build build-mcp
