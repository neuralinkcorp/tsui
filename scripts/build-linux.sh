#!/usr/bin/env bash

set -eu

source "$(dirname "$0")/_include.sh"

clean_artifacts
docker_build_linux linux/amd64 x86_64-linux linux-x86_64
echo
docker_build_linux linux/arm64 aarch64-linux linux-arm64
echo
echo "✅ Done! Artifacts are in $artifacts_dir"
echo
file "$artifacts_dir"/*
