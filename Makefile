# File: Makefile
# YggGram Cross-Platform Build System (Go C-Shared & Protobuf)

SHELL := /bin/bash
PROJECT_ROOT := $(shell pwd)
BUILD_DIR := $(PROJECT_ROOT)/build
CORE_CMD := ./core/cmd/libygggram

# Default NDK API Level
ANDROID_API ?= 24
ANDROID_NDK_HOME ?= $(ANDROID_NDK_ROOT)

.PHONY: all proto windows android android-arm64 android-x64 clean help

all: proto windows android

help:
	@echo "YggGram Build Targets:"
	@echo "  make proto          - Generate Protobuf v3 stubs for Go and Dart"
	@echo "  make windows        - Build ygggram.dll (Windows x64 via MinGW-w64)"
	@echo "  make android        - Build libygggram.so for arm64-v8a & x86_64"
	@echo "  make clean          - Clean all compilation artifacts"

proto:
	@echo "[+] Compiling protobuf definitions (proto/ygggram.proto)..."
	@mkdir -p core/pkg/yggproto app/lib/proto
	@if command -v protoc >/dev/null 2>&1; then \
		protoc --proto_path=proto \
			--go_out=core/pkg/yggproto --go_opt=paths=source_relative \
			--dart_out=app/lib/proto \
			proto/ygggram.proto; \
		echo "[✓] Protobuf code generation finished."; \
	else \
		echo "[!] Warning: 'protoc' compiler not found in PATH. Skipping code gen."; \
	fi

windows:
	@echo "[+] Building YggGram Windows x64 C-Shared Library (ygggram.dll)..."
	@mkdir -p $(BUILD_DIR)/windows/x64
	@mkdir -p app/windows/runner/assets
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
		go build -buildmode=c-shared \
		-ldflags="-s -w -X main.Version=1.0.0" \
		-o $(BUILD_DIR)/windows/x64/ygggram.dll $(CORE_CMD)
	@cp -f $(BUILD_DIR)/windows/x64/ygggram.dll app/windows/runner/assets/ 2>/dev/null || true
	@echo "[✓] Windows build successful: $(BUILD_DIR)/windows/x64/ygggram.dll"

android: android-arm64 android-x64

android-arm64:
	@echo "[+] Building YggGram Android arm64-v8a (libygggram.so)..."
	@bash scripts/build_android.sh arm64-v8a

android-x64:
	@echo "[+] Building YggGram Android x86_64 (libygggram.so)..."
	@bash scripts/build_android.sh x86_64

clean:
	@echo "[-] Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "[✓] Clean complete."
