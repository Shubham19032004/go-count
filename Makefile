VERSION ?= 1.0.0
GOCMD = go
GO_LDFLAGS = -ldflags="-X main.Version=$(VERSION)"
BIN_DIR = bin
BINARY_NAME = gocount

container:
	sudo /usr/local/go/bin/go run main.go run /bin/sh

ps:
	sudo $(GOCMD) run main.go ps

build:
	mkdir -p $(BIN_DIR)
	GOEXPERIMENT=cgocheck2 $(GOCMD) build $(GO_LDFLAGS) -o $(BIN_DIR)/$(BINARY_NAME) ./main.go

cm:
	sudo /usr/local/go/bin/go run main.go run --memory 100M --cpu "100 100000" /bin/sh

test-memory:
	@ROOTFS=$$(ls -dt /tmp/gocount/*/rootfs 2>/dev/null | head -1); \
	if [ -z "$$ROOTFS" ]; then \
		echo "No rootfs found. Run 'make container' first to create one, then re-run."; \
		exit 1; \
	fi; \
	echo "Copying test.py into $$ROOTFS ..."; \
	sudo cp test.py $$ROOTFS/test.py; \
	echo "Running container with 50M memory limit..."; \
	sudo /usr/local/go/bin/go run main.go run --memory 50M /usr/bin/python3 /test.py
