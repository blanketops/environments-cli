SHELL := /bin/bash
APP_NAME := blanketops-environments
BUILD_DIR := .
INSTALL_DIR := $(HOME)/.local/bin
FALLBACK_DIR := $(HOME)/bin

# Real binary output location — THIS is the only correct path.
BUILD_OUTPUT := bin/$(APP_NAME)

STATIC_OUTPUT := bin/$(APP_NAME)-static
GOFLAGS_STATIC := CGO_ENABLED=0 GOOS=linux GOARCH=amd64

.PHONY: build install uninstall static static-arm64 testexec clean

# ---------------------------------------
# Build
# ---------------------------------------
build:
	@echo "🔧 Building $(APP_NAME)..."
	@mkdir -p bin
	@go build -o $(BUILD_OUTPUT) $(BUILD_DIR)
	@echo "✅ Build complete: $(BUILD_OUTPUT)"
	@file $(BUILD_OUTPUT)

# ---------------------------------------
# Test if INSTALL_DIR allows execution
# ---------------------------------------
testexec:
	@echo "#!/bin/sh" > $(INSTALL_DIR)/.__exec_test
	@echo "echo test_ok" >> $(INSTALL_DIR)/.__exec_test
	@chmod +x $(INSTALL_DIR)/.__exec_test
	@$(INSTALL_DIR)/.__exec_test
	@rm -f $(INSTALL_DIR)/.__exec_test

# ---------------------------------------
# Install (auto-detect noexec)
# ---------------------------------------
install: build
	@echo "📦 Installing $(APP_NAME) into $(INSTALL_DIR)..."

	@if [ ! -d "$(INSTALL_DIR)" ]; then mkdir -p "$(INSTALL_DIR)"; fi

	# ALWAYS copy the REAL ELF binary from bin/
	@cp $(BUILD_OUTPUT) $(INSTALL_DIR)/$(APP_NAME)
	@chmod +x $(INSTALL_DIR)/$(APP_NAME)

	@echo "🔍 Testing executability of $(INSTALL_DIR)..."
	@if $(MAKE) --no-print-directory testexec >/dev/null 2>&1; then \
		echo "✅ Installed to $(INSTALL_DIR)/$(APP_NAME)"; \
	else \
		echo "⚠️  $(INSTALL_DIR) is mounted noexec — switching to $(FALLBACK_DIR)"; \
		mkdir -p "$(FALLBACK_DIR)"; \
		cp $(BUILD_OUTPUT) $(FALLBACK_DIR)/$(APP_NAME); \
		chmod +x $(FALLBACK_DIR)/$(APP_NAME); \
		echo "🎉 Installed to $(FALLBACK_DIR)/$(APP_NAME)"; \
		echo "ℹ️  Add to PATH: export PATH=\"$(FALLBACK_DIR):\$$PATH\""; \
	fi

# ---------------------------------------
# Uninstall
# ---------------------------------------
uninstall:
	@echo "🧹 Uninstalling $(APP_NAME)..."

	@echo "➡ Removing from user install dirs..."
	@rm -f "$(INSTALL_DIR)/$(APP_NAME)"
	@rm -f "$(FALLBACK_DIR)/$(APP_NAME)"

	@echo "➡ Removing from /usr/local/bin if exists..."
	@sudo rm -f /usr/local/bin/$(APP_NAME) || true

	@echo "➡ Removing built binaries from repo..."
	@rm -f bin/$(APP_NAME) bin/$(APP_NAME)-static bin/$(APP_NAME)-static-arm64

	@echo "✔️ All copies removed"
# ---------------------------------------
# Static Build (gokrazy-compatible)
# ---------------------------------------
static:
	@echo "🏗  Building static $(APP_NAME) for gokrazy..."
	@mkdir -p bin
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
		go build -trimpath -ldflags "-s -w" -o $(STATIC_OUTPUT) $(BUILD_DIR)
	@chmod +x $(STATIC_OUTPUT)
	@echo "🔎 Verifying static binary..."
	@ldd $(STATIC_OUTPUT) || true
	@echo "➡ Ready for gokrazy"

# ARM64 Static Build
static-arm64:
	@echo "🏗  Building static $(APP_NAME) for ARM64 gokrazy..."
	@mkdir -p bin
	@env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		go build -ldflags "-s -w" -o bin/$(APP_NAME)-static-arm64 $(BUILD_DIR)
	@chmod +x bin/$(APP_NAME)-static-arm64
	@echo "✅ Static ARM64 build complete"
	@file bin/$(APP_NAME)-static-arm64
	@echo "➡ Ready for gokrazy"

clean:
	@rm -rf bin
