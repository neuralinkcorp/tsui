# tsui

An elegant TUI for configuring [Tailscale](https://tailscale.com/).

We built this because, while Tailscale has lovely desktop apps for macOS and Windows, Linux users are stuck configuring Tailscale with CLI commands. Some of tsui's features are:

- Edit Tailscale options with a full settings interface
- Switch exit nodes and compare their latency
- View and copy debug information
- See your bandwidth
- Easily log in, out, and reauthenticate

<img width="1039" alt="Screenshot of tsui" src="https://github.com/user-attachments/assets/9768a858-027a-431b-a5e2-39457d508648">

## Installation

We support tsui on macOS and Linux, both x86_64 and aarch64 architectures.

Run our installer:

```sh
curl -sS https://tsui.neuralink.com/install | bash
```

You can also download the latest tsui release from the [releases page](https://github.com/neuralinkcorp/tsui/releases/latest). We distribute tsui as a single binary that shouldn't require any dependencies.

## Development

There are a couple ways to develop and build tsui, depending on what exactly your goals are.

### With Nix

If you have [Nix](https://nixos.org/) installed, this is likely your best option. It will guarantee your environment is set up consistently and the dependencies you need are available.

Make sure you have the `nix-command` and `flakes` experimental features enabled to use Nix flakes.

Develop with a dev shell:

```sh
nix develop

go run .
```

Run directly through Nix:

```sh
nix run
```

Build a binary for your platform:

```sh
nix build

./result/bin/tsui
```

### With Go

If you want to use Nix, you can still build with the Go toolchain. you will need Go installed and, on Linux, `libx11-dev`. On macOS, you may also need the XCode command line tools.

Develop:

```sh
go run .
```

Build a binary for your platform:

```sh
go build .

./tsui
```

## Production Builds

We provide scripts to generate cross-platform builds equivalent to those distributed on our [releases page](https://github.com/neuralinkcorp/tsui/releases/latest).

Prerequisites:

- [Docker](https://www.docker.com/)
- [Nix](https://nixos.org/) (only on macOS)

Build binaries for all platforms and architectures (only works on macOS):

```sh
./scripts/build-all.sh
```

Build Linux binaries for all architectures:

```sh
./scripts/build-linux.sh
```

For either script, binaries will be outputted to the `artifacts/` directory.
