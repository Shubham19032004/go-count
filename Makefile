VERSION ?= 1.0.0
GOCMD = go
GO_LDFLAGS = -ldflags="-X main.Version=$(VERSION)"
BIN_DIR = bin
BINARY_NAME = gocount

container:
	sudo /usr/local/go/bin/go run main.go run /bin/bash

ps:
	sudo $(GOCMD) run main.go ps

build:
	mkdir -p $(BIN_DIR)
	GOEXPERIMENT=cgocheck2 $(GOCMD) build $(GO_LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./main.go
