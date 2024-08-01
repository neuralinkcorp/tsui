#!/usr/bin/env bash

# Redirects from https://neuralink.com/tsui/install.sh

set -eu

install_dir="/usr/local/bin"
install_path="$install_dir/tsui"
repo_url="https://github.com/neuralinkcorp/tsui"

case $(uname -ms) in
'Darwin x86_64')
  target=macos-x86_64
  ;;
'Darwin arm64')
  target=macos-aarch64
  ;;
'Linux x86_64')
  target=linux-x86_64
  ;;
'Linux aarch64' | 'Linux arm64')
  target=linux-aarch64
  ;;
*)
  echo "😦 Sorry, we don't have binaries for your platform: '$(uname -ms)'"
  echo "   You can build from source or submit an issue at $repo_url/issues"
  exit 1
  ;;
esac

if [[ $# = 0 ]]; then
  echo "📦 Downloading latest tsui release to $install_path"
  download_url=$repo_url/releases/latest/download/tsui-$target
else
  echo "🛠️  Downloading specified tsui version (v$1) to $install_path"
  download_url=$repo_url/releases/download/v$1/tsui-$target
fi

sudo curl --fail --location --progress-bar --output "$install_path" "$download_url" ||
  (echo "🚨 Failed to download tsui from '$download_url'" &&
    echo "   You can download the latest release yourself from $repo_url/releases/latest" &&
    exit 1)

sudo chmod +x "$install_path"

if [[ $target == linux-* ]]; then
  start_command="sudo tsui"
else
  start_command="tsui"
fi

echo "🎉 Installed tsui! Run '$start_command' to configure Tailscale."
