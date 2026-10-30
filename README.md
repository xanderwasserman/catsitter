# Catsitter

A little cat that sits on your computer and keeps it awake. It holds a sleep inhibitor the whole time it is on duty. Once you have been idle, it walks across the keyboard: a tiny mouse jiggle and a Shift tap. The cat paws at the keys when it does.

## Install

macOS and Linux:

```sh
curl -fsSL https://github.com/xanderwasserman/catsitter/releases/latest/download/install.sh | sh
```

Windows (PowerShell):

```powershell
irm https://github.com/xanderwasserman/catsitter/releases/latest/download/install.ps1 | iex
```

Nothing needs sudo or an administrator. The script picks the archive for this machine from the latest GitHub release, checks it against `checksums.txt`, installs it, and starts Catsitter. Run the same command again later to update. If that version is already installed, it leaves the app alone. If Catsitter is running, the script stops it, replaces the files, and starts it again. On Linux it prints the `PATH` line to add when `~/.local/bin` is missing from `PATH`.

A pipe does not forward arguments. Pin a version or uninstall like this:

```sh
curl -fsSL https://github.com/xanderwasserman/catsitter/releases/latest/download/install.sh | sh -s -- --version 0.1.0
curl -fsSL https://github.com/xanderwasserman/catsitter/releases/latest/download/install.sh | sh -s -- --uninstall
```

```powershell
iex "& { $(irm https://github.com/xanderwasserman/catsitter/releases/latest/download/install.ps1) } -Version 0.1.0"
iex "& { $(irm https://github.com/xanderwasserman/catsitter/releases/latest/download/install.ps1) } -Uninstall"
```

`--uninstall` is accepted too. Uninstall removes the app and leaves your settings file where it is.

| System | Location |
| --- | --- |
| macOS | `~/Applications/Catsitter.app` (quarantine flag cleared) |
| Linux | `~/.local/bin/catsitter`, plus a desktop entry and icon |
| Windows | `%LOCALAPPDATA%\Programs\Catsitter`, plus a Start Menu shortcut |

On Linux, `~/.local/bin` needs to be on `PATH`.

## Usage

Use the menu-bar or system-tray icon, or right-click the cat. Both can pause and resume, toggle each action, pick how long to wait, pick how long to keep watch, hide the cat while you work, and quit.

Drag the cat to move it. It remembers the spot.

Defaults: sleep inhibitor on, mouse jiggle on, Shift tap on, window switch off, idle interval 10 seconds, and it runs until you stop it. Nudges wait until you have been idle for that interval. The inhibitor stays held the whole time, including while you are active. Pause freezes the timer and lets the machine sleep. Quit releases the inhibitor.

```sh
catsitter --seconds 15 --pixels 2 --duration 2h
catsitter --mode mk --tabs 0 --key alt
catsitter --no-pet
catsitter --version
```

Command-line flags last for that run. The tray and the cat menu are what get written to the settings file.

Window switch is Cmd+Tab on macOS and Alt+Tab elsewhere. `--tabs 0` switches a random one to three windows. Duration accepts `10m`, `2h`, `1.5h`, `2h30m`, `90s`, and the longer names (`minutes`, `hours`, `seconds`). `unlimited`, `forever`, and `until I stop it` mean no limit.

Settings live in the OS config directory under `catsitter/config.json`.

## Permissions

macOS nudges need Accessibility permission. Catsitter explains that and opens the Accessibility pane. Until you allow it, the cat still keeps the display awake and skips the jiggle and the key tap. The download is unsigned. If Gatekeeper stops it, right-click Catsitter, choose Open, then Open Anyway (or allow it under Privacy & Security). The installer clears the quarantine bit.

Windows SmartScreen may say the app is unrecognized. Choose More info, then Run anyway.

Linux nudges need an X11 session, or XWayland, because they use the XTEST extension. On a pure Wayland session the sleep inhibitor can still work. Install the usual client libraries if they are missing: `libgl1`, `libx11-6`, `libxrandr2`, `libxinerama1`, `libxcursor1`, `libxi6`, and `libxxf86vm1`.

## Why Ebitengine

The pet is a small borderless window, always on top, with a transparent background. [Ebitengine](https://ebitengine.org) draws that at a low tick rate and does it without CGO on Linux and Windows. The pet is a small interface, so another animal can be added later. This release only ships the cat, and the sprite sheet is original pixel art.

## Build

Go 1.25 or newer.

```sh
go test ./...
go build -o catsitter ./cmd/catsitter
```

macOS builds need CGO for IOKit, CoreGraphics, and the menu bar. Linux and Windows build with `CGO_ENABLED=0`, including cross-compiles:

```sh
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o catsitter.exe ./cmd/catsitter
```

GoReleaser builds the six release archives and `checksums.txt`. Tag `v0.1.0` (or any `v*`) to publish them, with `install.sh` and `install.ps1` attached to the release.
