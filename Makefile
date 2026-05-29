BINARY := adb-link
PKG := github.com/gnodux/adb-link
CMD := ./cmd/adb-link
BUILD_DIR := bin
VERSION ?= 0.1.0
LDFLAGS := -X main.version=$(VERSION) -s -w

.PHONY: all build run-all run-api run-mcp test tidy fmt vet lint clean install

all: build

build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(CMD)

install:
	go install -ldflags "$(LDFLAGS)" $(CMD)

run-all: build
	$(BUILD_DIR)/$(BINARY) run-all

run-api: build
	$(BUILD_DIR)/$(BINARY) run-api

run-mcp: build
	$(BUILD_DIR)/$(BINARY) run-mcp

test:
	go test ./... -count=1

tidy:
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet

clean:
	rm -rf $(BUILD_DIR)

# Cross-compile helpers
build-linux:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64 $(CMD)

build-darwin:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 $(CMD)

build-windows:
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe $(CMD)
