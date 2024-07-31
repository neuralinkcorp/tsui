#!/usr/bin/env bash

repo=$(git rev-parse --show-toplevel)
export repo

export artifacts_dir="$repo/artifacts"

# Usage: clean_artifacts
function clean_artifacts {
  mkdir -p "$artifacts_dir"
  rm -f "$artifacts_dir"/*
}

# Usage: build_macos <nix_platform> <tsui_platform>
function build_macos {
  echo "🔨 Building: $2..."
  nix build ".#defaultPackage.$1"
  cp ./result/bin/tsui "$artifacts_dir/tsui-$2"
}

# Usage: docker_build_linux <docker_platform> <nix_platform> <tsui_platform>
function docker_build_linux {
  echo "🔨 Building: $3..."

  name="tsui-linux-build-$2"

  docker build \
    --platform "$1" \
    --tag "$name" \
    --file "$repo/Dockerfile.build" \
    "$repo"

  # Clean up any earlier failed containers.
  docker rm "$name" 2>/dev/null || true

  # Using --tty so Nix's output is prettier.
  docker run \
    --tty \
    --name "$name" \
    --platform "$1" \
    -v "$repo:/opt/tsui" \
    --workdir /opt/tsui \
    "$name" \
    nix build ".#defaultPackage.$2"
  
  docker cp \
    "$name:/opt/tsui/result/bin/tsui" \
    "$artifacts_dir/tsui-$3"

  docker rm "$name"
}
