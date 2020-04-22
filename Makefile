all: build

build:
	@go build

test:
	go test ./...

install:
	@go install

clean:
	@go clean

.PHONY: all build test install
