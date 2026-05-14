GO ?= go
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
VERSION ?= 0.1.8
LD_FLAGS_BASE ?= -s -w -extldflags '-static'
BUILD_FLAGS ?= -trimpath -a -ldflags "$(LD_FLAGS_BASE)"
BIN_DIR ?= bin
BINARY ?= $(BIN_DIR)/FastPVE
CMD ?= ./cmd/fastpve
DOWNLOAD_CMD ?= ./cmd/download
WORKFILE ?= go.work
WORKFILE_ABS := $(abspath $(WORKFILE))
RELEASE_BINARY ?= $(BIN_DIR)/FastPVE-$(VERSION)
VERSION_FILE ?= $(BIN_DIR)/version.txt
BINARY_DOWNLOAD ?= $(BIN_DIR)/fastpve-download

.PHONY: all build build-remote download download-remote clean release validate-version

all: build

# Build without HAS_REMOTE_URL; ignore go.work so the default go.mod is used.
build: $(BIN_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOWORK=off $(GO) build $(BUILD_FLAGS) -o $(BINARY) $(CMD)

# Build with HAS_REMOTE_URL; use go.work so the replace for remote cache is applied.
build-remote: $(BIN_DIR) $(WORKFILE)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOWORK=$(WORKFILE_ABS) $(GO) build $(BUILD_FLAGS) -tags HAS_REMOTE_URL -o $(BINARY) $(CMD)

# Build download-only CLI without HAS_REMOTE_URL.
download: $(BIN_DIR)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOWORK=off $(GO) build $(BUILD_FLAGS) -o $(BINARY_DOWNLOAD) $(DOWNLOAD_CMD)

# Build download-only CLI with HAS_REMOTE_URL.
download-remote: $(BIN_DIR) $(WORKFILE)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOWORK=$(WORKFILE_ABS) $(GO) build $(BUILD_FLAGS) -tags HAS_REMOTE_URL -o $(BINARY_DOWNLOAD) $(DOWNLOAD_CMD)

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

clean:
	rm -f $(BINARY) $(BINARY_DOWNLOAD) $(VERSION_FILE) $(BIN_DIR)/FastPVE-*

release: validate-version $(VERSION_FILE)

validate-version:
	@if [ -z "$(strip $(VERSION))" ]; then \
		echo "VERSION is required (e.g. VERSION=0.1.5)"; \
		exit 1; \
	fi

$(RELEASE_BINARY): $(BIN_DIR) $(WORKFILE)
	GOOS=$(GOOS) GOARCH=$(GOARCH) GOWORK=$(WORKFILE_ABS) $(GO) build -trimpath -a -ldflags "$(LD_FLAGS_BASE) -X main.version=$(VERSION)" -tags HAS_REMOTE_URL -o $@ $(CMD)

$(VERSION_FILE): $(RELEASE_BINARY)
	@echo "VERSION=$(VERSION)" > $@
	@echo "" >> $@
	@echo "FASTPVE_SHA256=$$(sha256sum $(RELEASE_BINARY) | awk '{print $$1}')" >> $@
