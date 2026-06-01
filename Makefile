.PHONY: all build test install clean dev build-all

BINARY := porttidy
CMD := ./cmd/porttidy

all: build

build:
	go build -o $(BINARY) $(CMD)

dev:
	go run $(CMD) scan

test:
	go test ./...

install:
	go install $(CMD)

clean:
	rm -f $(BINARY)
	rm -rf dist
	go clean

# Cross-compilation
build-all:
	mkdir -p dist
	GOOS=darwin  GOARCH=arm64 go build -o dist/porttidy-darwin-arm64 $(CMD)
	GOOS=darwin  GOARCH=amd64 go build -o dist/porttidy-darwin-amd64 $(CMD)
	GOOS=linux   GOARCH=amd64 go build -o dist/porttidy-linux-amd64 $(CMD)
	GOOS=linux   GOARCH=arm64 go build -o dist/porttidy-linux-arm64 $(CMD)
	GOOS=windows GOARCH=amd64 go build -o dist/porttidy-windows-amd64.exe $(CMD)
