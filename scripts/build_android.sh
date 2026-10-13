#!/usr/bin/env bash
# File: scripts/build_android.sh
# Builds YggGram Go core into libygggram.so for Android using Android NDK toolchain

set -euo pipefail

TARGET_ABI="${1:-arm64-v8a}"
ANDROID_API_LEVEL="${ANDROID_API:-24}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=========================================================="
echo "  YggGram Android Core Compiler (libygggram.so)"
echo "=========================================================="
echo "[*] Target ABI: ${TARGET_ABI}"
echo "[*] Android API Level: ${ANDROID_API_LEVEL}"

# Locate Android NDK
if [[ -z "${ANDROID_NDK_HOME:-}" && -z "${ANDROID_NDK_ROOT:-}" ]]; then
    if [[ -d "${ANDROID_HOME:-}/ndk" ]]; then
        # Pick the latest installed NDK version
        LATEST_NDK=$(ls -d "${ANDROID_HOME}/ndk/"* 2>/dev/null | sort -V | tail -n 1)
        if [[ -n "${LATEST_NDK}" ]]; then
            export ANDROID_NDK_HOME="${LATEST_NDK}"
        fi
    fi
fi

NDK_PATH="${ANDROID_NDK_HOME:-${ANDROID_NDK_ROOT:-}}"
if [[ -z "${NDK_PATH}" || ! -d "${NDK_PATH}" ]]; then
    echo "[!] Error: Android NDK not found. Please set ANDROID_NDK_HOME or ANDROID_NDK_ROOT."
    exit 1
fi

echo "[*] Using NDK: ${NDK_PATH}"

# Detect host OS for prebuilt toolchain
HOST_OS="linux-x86_64"
if [[ "$OSTYPE" == "darwin"* ]]; then
    HOST_OS="darwin-x86_64"
fi

TOOLCHAIN="${NDK_PATH}/toolchains/llvm/prebuilt/${HOST_OS}"
if [[ ! -d "${TOOLCHAIN}" ]]; then
    echo "[!] Error: LLVM toolchain directory not found at ${TOOLCHAIN}"
    exit 1
fi

# Configure target architecture mapping
case "${TARGET_ABI}" in
    "arm64-v8a")
        GOARCH="arm64"
        TARGET_TRIPLE="aarch64-linux-android"
        ;;
    "x86_64")
        GOARCH="amd64"
        TARGET_TRIPLE="x86_64-linux-android"
        ;;
    "armeabi-v7a")
        GOARCH="arm"
        TARGET_TRIPLE="armv7a-linux-androideabi"
        export GOARM="7"
        ;;
    *)
        echo "[!] Unsupported ABI: ${TARGET_ABI}. Supported: arm64-v8a, x86_64, armeabi-v7a"
        exit 1
        ;;
esac

CC_BIN="${TOOLCHAIN}/bin/${TARGET_TRIPLE}${ANDROID_API_LEVEL}-clang"
CXX_BIN="${TOOLCHAIN}/bin/${TARGET_TRIPLE}${ANDROID_API_LEVEL}-clang++"

if [[ ! -f "${CC_BIN}" ]]; then
    echo "[!] Error: Clang cross-compiler not found: ${CC_BIN}"
    exit 1
fi

BUILD_OUT="${PROJECT_ROOT}/build/android/${TARGET_ABI}"
JNILIBS_TARGET="${PROJECT_ROOT}/app/android/app/src/main/jniLibs/${TARGET_ABI}"

mkdir -p "${BUILD_OUT}"
mkdir -p "${JNILIBS_TARGET}"

OUTPUT_SO="${BUILD_OUT}/libygggram.so"

echo "[*] Compiling ${OUTPUT_SO} with CGO..."
cd "${PROJECT_ROOT}"

CGO_ENABLED=1 \
GOOS=android \
GOARCH="${GOARCH}" \
CC="${CC_BIN}" \
CXX="${CXX_BIN}" \
go build -buildmode=c-shared \
    -ldflags="-s -w -X main.Version=1.0.0-android -extldflags '-Wl,-soname,libygggram.so'" \
    -o "${OUTPUT_SO}" \
    ./core/cmd/libygggram

echo "[✓] Built: ${OUTPUT_SO}"

# Deploy to Flutter Android jniLibs
cp -f "${OUTPUT_SO}" "${JNILIBS_TARGET}/libygggram.so"
echo "[✓] Copied to Flutter jniLibs: ${JNILIBS_TARGET}/libygggram.so"
