.PHONY: build test vet install clean

build:
	go build -o bin/aruba-central-pp-cli ./cmd/aruba-central-pp-cli

test:
	go test ./...

vet:
	go vet ./...

install:
	go install ./cmd/aruba-central-pp-cli

clean:
	rm -rf bin/
