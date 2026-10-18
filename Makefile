# File: Makefile
# YggGram Cross-Platform Build System (Go C-Shared & Protobuf)

SHELL := /bin/bash
PROJECT_ROOT := $(shell pwd)
BUILD_DIR := $(PROJECT_ROOT)/build
CORE_CMD := ./core/cmd/libygggram

# Default NDK API Level
ANDROID_API ?= 24
ANDROID_NDK_HOME ?= $(ANDROID_NDK_ROOT)

.PHONY: all proto windows android android-arm64 android-x64 clean check help

all: proto windows android

help:
	@echo "YggGram Build Targets:"
	@echo "  make proto          - Генерация Go- и Dart-биндингов protobuf (buf + protoc-gen-go + protoc-gen-dart)"
	@echo "  make check          - go vet + go test + flutter analyze + flutter test (как в CI)"
	@echo "  make windows        - Build ygggram.dll (Windows x64 via MinGW-w64)"
	@echo "  make android        - Build libygggram.so for arm64-v8a & x86_64"
	@echo "  make clean          - Clean all compilation artifacts"

check:
	cd core && go vet ./... && go test ./...
	cd app && flutter analyze --fatal-infos && flutter test

proto:
	@echo "[+] Generating Go and Dart bindings from proto/ygggram.proto..."
	@command -v buf >/dev/null 2>&1 || { echo "[!] 'buf' not found: go install github.com/bufbuild/buf/cmd/buf@latest"; exit 1; }
	@command -v protoc-gen-go >/dev/null 2>&1 || { echo "[!] 'protoc-gen-go' not found: go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.34.2"; exit 1; }
	@command -v protoc-gen-dart >/dev/null 2>&1 || { echo "[!] 'protoc-gen-dart' not found: dart pub global activate protoc_plugin"; exit 1; }
	@buf generate
	@echo "[✓] core/proto/ygggram.pb.go и app/lib/gen/yggproto/*.dart обновлены"

windows:
	@echo "[+] Building YggGram Windows x64 C-Shared Library (ygggram.dll)..."
	@mkdir -p $(BUILD_DIR)/windows/x64
	@mkdir -p app/windows/runner/assets
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
		go build -buildmode=c-shared \
		-ldflags="-s -w -checklinkname=0 -X main.Version=1.0.0" \
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
