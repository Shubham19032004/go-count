VERSION ?= 1.0.0
GO = /usr/local/go/bin/go
GO_LDFLAGS = -ldflags="-X main.Version=$(VERSION)"
BIN_DIR = bin
BINARY_NAME = gocount

build:
	mkdir -p $(BIN_DIR)
	GOEXPERIMENT=cgocheck2 $(GO) build $(GO_LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./main.go

run:
	sudo $(GO) run main.go run /bin/sh

run-memory:
	sudo $(GO) run main.go run --memory 100M --cpu "100 100000" /bin/sh

ps:
	sudo $(GO) run main.go ps

stop:
	@if [ -z "$(ID)" ]; then echo "Usage: make stop ID=<container_id>"; exit 1; fi
	sudo $(GO) run main.go stop $(ID)

start:
	@if [ -z "$(ID)" ]; then echo "Usage: make start ID=<container_id>"; exit 1; fi
	sudo $(GO) run main.go start $(ID)

rm:
	@if [ -z "$(ID)" ]; then echo "Usage: make rm ID=<container_id>"; exit 1; fi
	sudo $(GO) run main.go rm $(ID)

inspect:
	@if [ -z "$(ID)" ]; then echo "Usage: make inspect ID=<container_id>"; exit 1; fi
	sudo $(GO) run main.go inspect $(ID)

test-memory:
	@ROOTFS=$$(ls -dt /tmp/gocount/*/rootfs 2>/dev/null | head -1); \
	if [ -z "$$ROOTFS" ]; then \
		echo "No rootfs found. Run 'make run' first to create one, then re-run."; \
		exit 1; \
	fi; \
	echo "Copying test.py into $$ROOTFS ..."; \
	sudo cp test.py $$ROOTFS/test.py; \
	echo "Running container with 50M memory limit..."; \
	sudo $(GO) run main.go run --memory 50M /usr/bin/python3 /test.py

clean:
	rm -rf $(BIN_DIR)

.PHONY: build run run-memory ps stop start rm inspect test-memory clean
