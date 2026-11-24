.PHONY: all build build-windows build-all test clean

BINARY_NAME=tabwrangler
VERSION=1.0.1

all: test build

build:
	go build -o $(BINARY_NAME) ./cmd/tabwrangler

build-windows:
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_NAME).exe ./cmd/tabwrangler

build-all:
	@mkdir -p bin
	GOOS=darwin  GOARCH=arm64 go build -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/tabwrangler
	GOOS=darwin  GOARCH=amd64 go build -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/tabwrangler
	GOOS=linux   GOARCH=amd64 go build -o bin/$(BINARY_NAME)-linux-amd64  ./cmd/tabwrangler
	GOOS=linux   GOARCH=arm64 go build -o bin/$(BINARY_NAME)-linux-arm64  ./cmd/tabwrangler
	GOOS=windows GOARCH=amd64 go build -o bin/$(BINARY_NAME).exe         ./cmd/tabwrangler

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf bin outputs
