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

# -------- 开发部署（本地编译 → 推送到 PVE 测试机）--------

PVE_HOST ?= 100.72.34.32
PVE_BIN_PATH ?= /usr/local/bin/fastpve
SSH_KEY ?= $(HOME)/.ssh/id_rsa

# 交叉编译 linux/amd64 + scp 推送（-C 启用压缩传输）
deploy-pve: $(BIN_DIR)
	GOOS=linux GOARCH=amd64 GOWORK=off $(GO) build $(BUILD_FLAGS) -o $(BIN_DIR)/FastPVE-linux $(CMD)
	scp -C -i $(SSH_KEY) "$(BIN_DIR)/FastPVE-linux" root@$(PVE_HOST):$(PVE_BIN_PATH)
	ssh -i $(SSH_KEY) root@$(PVE_HOST) "chmod +x $(PVE_BIN_PATH) && echo '部署完成:'; fastpve version"
	@echo ""
	@echo "  部署成功！现在可以 SSH 到 PVE 测试："
	@echo "    ssh root@$(PVE_HOST)"
	@echo "    fastpve"
	@echo "    fastpve ai \"查看系统状态\""

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
