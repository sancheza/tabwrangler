.PHONY: all build build-windows package-windows build-all test clean

BINARY_NAME=tabwrangler
VERSION=1.0.13

all: test build

build:
	go build -trimpath -ldflags="-s -w" -o $(BINARY_NAME) ./cmd/tabwrangler

build-windows:
	@mkdir -p bin
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/$(BINARY_NAME).exe ./cmd/tabwrangler

package-windows: build-windows
	@mkdir -p /tmp/tw-win-pkg
	@rm -rf /tmp/tw-win-pkg/*
	@cp bin/$(BINARY_NAME).exe /tmp/tw-win-pkg/
	@cp -r extension /tmp/tw-win-pkg/
	@rm -f bin/$(BINARY_NAME)-windows-amd64.zip
	@(cd /tmp/tw-win-pkg && zip -r $(CURDIR)/bin/$(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME).exe extension)
	@rm -rf /tmp/tw-win-pkg

build-all:
	@mkdir -p bin
	GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/tabwrangler
	GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/tabwrangler
	GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/$(BINARY_NAME)-linux-amd64  ./cmd/tabwrangler
	GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bin/$(BINARY_NAME)-linux-arm64  ./cmd/tabwrangler
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o bin/$(BINARY_NAME).exe         ./cmd/tabwrangler

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf bin outputs
