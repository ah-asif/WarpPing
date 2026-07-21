BINARY     := warpping
CMD_DIR    := ./cmd/warpping
INSTALL_DIR := /usr/local/bin

.PHONY: all build run install uninstall clean vet fmt tidy allow-unprivileged-ping

all: build

## build: compile the binary into ./bin/warpping
build:
	mkdir -p bin
	go build -o bin/$(BINARY) $(CMD_DIR)

## run: build and run warpping with default settings
run: build
	./bin/$(BINARY)

## install: build and copy the binary to /usr/local/bin (may need sudo)
install: build
	install -m 0755 bin/$(BINARY) $(INSTALL_DIR)/$(BINARY)

## uninstall: remove the installed binary
uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)

## clean: remove build artifacts
clean:
	rm -rf bin

## vet: run go vet across the module
vet:
	go vet ./...

## fmt: format all Go source files
fmt:
	gofmt -w $(shell find . -name '*.go')

## tidy: sync go.mod/go.sum with imports
tidy:
	go mod tidy

## allow-unprivileged-ping: let warpping send ICMP pings without root/sudo
allow-unprivileged-ping:
	sudo sysctl -w net.ipv4.ping_group_range="0 2147483647"
