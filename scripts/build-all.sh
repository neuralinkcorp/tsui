#!/usr/bin/env bash

set -eu

source "$(dirname "$0")/_include.sh"

if [[ $(uname -s) != "Darwin" ]]; then
  echo "🚨 The unified build script only works on macOS."
  echo "   You can run build-linux.sh to build only for Linux."
  exit 1
fi

clean_artifacts
build_macos aarch64-darwin macos-aarch64
echo
build_macos aarch64-darwin macos-x86_64
echo
docker_build_linux linux/arm64 aarch64-linux linux-arm64
echo
docker_build_linux linux/amd64 x86_64-linux linux-x86_64
echo
echo "✅ Done! Artifacts are in $artifacts_dir"
echo
file "$artifacts_dir"/*
