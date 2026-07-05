# Substitute Soccer — Go port

[![Latest release](https://img.shields.io/github/v/release/chrplr/soccer-go)](https://github.com/chrplr/soccer-go/releases/latest)

**▶ Play it in your browser: <https://chrplr.github.io/soccer-go/>**

A Go re-implementation of the Pygame Zero game **Substitute Soccer** from *Code the Classics
Volume 1* (Raspberry Pi Press), built on
[go-sdl3](https://github.com/Zyko0/go-sdl3) and the
[pgzgo](https://github.com/chrplr/pgzgo) harness.

All images, sounds and music are embedded, so `go build` produces a single
self-contained binary that needs no asset files at run time. Keyboard and gamepad
are both supported.

## Download

Prebuilt, self-contained binaries — no install, no dependencies, assets embedded.
Grab the latest for your platform:

- **Linux** (amd64) — [soccer-linux-amd64.tar.gz](https://github.com/chrplr/soccer-go/releases/latest/download/soccer-linux-amd64.tar.gz)
- **macOS** (Apple Silicon) — [soccer-macos-arm64.tar.gz](https://github.com/chrplr/soccer-go/releases/latest/download/soccer-macos-arm64.tar.gz)
- **Windows** (amd64) — [soccer-windows-amd64.zip](https://github.com/chrplr/soccer-go/releases/latest/download/soccer-windows-amd64.zip)

All versions are on the [releases page](https://github.com/chrplr/soccer-go/releases).

## Run

```sh
go run .
```

go-sdl3 bundles the SDL3, SDL3_image and SDL3_mixer libraries and extracts them at
startup, so no system SDL install is needed.

## Provenance & license

Ported to Go from the Python original in *Code the Classics Volume 1*. The game
design and original assets are © their respective authors / Raspberry Pi Press —
add the appropriate license before redistributing.

The original Python code and assets are in Raspberry Pi Press's [Code the Classics — Volume 1](https://github.com/raspberrypipress/Code-the-Classics-Vol1) repository.

The Go source code of this port is distributed under the MIT License.

See `Python_and_Go_implementation_comparison.md` for a walkthrough of how the port
maps onto the original.
