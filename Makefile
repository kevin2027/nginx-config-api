# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

# Binary names
BINARY_NAME=nginx-config-api

# Versioning
VERSION=1.0.0
BUILD_TIME=`date +%FT%T%z`

# Platform specific variables
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	# macOS
	OS_NAME=darwin
else ifeq ($(OS),Windows_NT)
	# Windows
	OS_NAME=windows
else
	# Linux
	OS_NAME=linux
endif

# Architecture specific variables
ARCH := $(shell uname -m)
ifeq ($(ARCH),armv7l)
	# ARMv7
	ARCH_NAME=arm
	ARCH_FLAGS=GOARCH=arm GOARM=7
else ifeq ($(ARCH),aarch64)
	# ARM64
	ARCH_NAME=arm64
	ARCH_FLAGS=GOARCH=arm64
else
	# AMD64
	ARCH_NAME=amd64
	ARCH_FLAGS=GOARCH=amd64
endif

# Build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

help:
	@echo "Available targets:"
	@echo "  make           : Build the project"
	@echo "  make clean     : Clean up the project"
	@echo "  make run       : Build and run the project"
	@echo "  make test      : Run tests"
	@echo "  make deps      : Install project dependencies"
	@echo "  make build-linux-amd64   : Build for Linux AMD64"
	@echo "  make build-linux-arm     : Build for Linux ARM"
	@echo "  make build-linux-arm64   : Build for Linux ARM64"
	@echo "  make build-windows-amd64 : Build for Windows AMD64"
	@echo "  make build-darwin-amd64  : Build for macOS AMD64"
	@echo "  make build-all            : Build for all platforms"

all: clean build

build:
	$(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME) -v

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

run:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) -v ./...
	./$(BINARY_NAME)

test:
	$(GOTEST) -v ./...

deps:
	$(GOGET) github.com/gorilla/mux
	$(GOGET) github.com/kardianos/service
	$(GOGET) github.com/sirupsen/logrus
	$(GOGET) gopkg.in/natefinch/lumberjack.v2

build-linux-amd64:
	$(ARCH_FLAGS) GOOS=linux $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)_linux_$(ARCH_NAME) -v

build-linux-arm:
	GOOS=linux GOARCH=arm $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)_linux_arm -v

build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)_linux_arm64 -v

build-windows-amd64:
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)_windows_amd64.exe -v

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)_darwin_amd64 -v

build-all: build-linux-amd64 build-linux-arm build-linux-arm64 build-windows-amd64 build-darwin-amd64
