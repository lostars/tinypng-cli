BINARY_NAME=qbit
LDFLAGS=-ldflags "-X main.Version=dev -w -s"

.PHONY: build deps

build:
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) main.go

deps:
	go mod tidy
