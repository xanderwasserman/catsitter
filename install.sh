#!/bin/sh
# Install Catsitter for the current user. No sudo.
#
#   curl -fsSL https://github.com/xanderwasserman/catsitter/releases/latest/download/install.sh | sh
#   curl -fsSL .../install.sh | sh -s -- --version 0.1.0
#   curl -fsSL .../install.sh | sh -s -- --uninstall
set -eu

REPO="${CATSITTER_REPO:-xanderwasserman/catsitter}"
RELEASE_BASE="${CATSITTER_RELEASE_BASE:-}"

main() {
die() {
	echo "catsitter install: $*" >&2
	exit 1
}

usage() {
	echo "Usage: install.sh [--version VERSION] [--uninstall]"
	echo "Installs Catsitter for the current user. Re-run the script to update."
}

want_version=""
uninstall=0
while [ $# -gt 0 ]; do
	case "$1" in
	--help | -h)
		usage
		exit 0
		;;
	--uninstall)
		uninstall=1
		;;
	--version)
		[ $# -ge 2 ] || die "--version needs a value"
		shift
		want_version=$1
		;;
	--version=*)
		want_version=${1#--version=}
		;;
	*)
		die "unknown argument: $1"
		;;
	esac
	shift
done

case "$(uname -s)" in
Linux) goos=linux ;;
Darwin) goos=darwin ;;
*) die "unsupported operating system: $(uname -s)" ;;
esac

case "$(uname -m)" in
x86_64 | amd64) goarch=amd64 ;;
aarch64 | arm64) goarch=arm64 ;;
*) die "unsupported architecture: $(uname -m)" ;;
esac

if [ "$goos" = "darwin" ]; then
	install_bin="$HOME/Applications/Catsitter.app/Contents/MacOS/catsitter"
else
	install_bin="$HOME/.local/bin/catsitter"
fi

strip_v() {
	printf '%s' "$1" | tr -d ' \t\r\n' | sed 's/^v//'
}

installed_version() {
	if [ ! -f "$install_bin" ]; then
		return 0
	fi
	out=$("$install_bin" --version 2>/dev/null || true)
	strip_v "$out"
}

running() {
	command -v pgrep >/dev/null 2>&1 && pgrep -x catsitter >/dev/null 2>&1
}

stop_running() {
	command -v pkill >/dev/null 2>&1 || die "pkill is required to replace a running Catsitter"
	pkill -x catsitter >/dev/null 2>&1 || true
	i=0
	while running; do
		i=$((i + 1))
		if [ "$i" -gt 50 ]; then
			die "Catsitter is still running"
		fi
		sleep 0.1
	done
}

start_app() {
	# A subshell so the pet survives this script exiting.
	(
		nohup "$install_bin" >/dev/null 2>&1 &
	)
}

warn_path() {
	[ "$goos" = "linux" ] || return 0
	case ":$PATH:" in
	*":$HOME/.local/bin:"* | *":$HOME/.local/bin/:"*) return 0 ;;
	esac
	echo "catsitter install: ~/.local/bin is not on PATH. Add this line to your shell profile:" >&2
	# shellcheck disable=SC2016
	echo 'export PATH="$HOME/.local/bin:$PATH"' >&2
}

remove_install() {
	if [ "$goos" = "darwin" ]; then
		rm -rf "$HOME/Applications/Catsitter.app"
	else
		rm -f "$install_bin"
		rm -f "$HOME/.local/share/applications/catsitter.desktop"
		rm -f "$HOME/.local/share/icons/hicolor/64x64/apps/catsitter.png"
	fi
}

if [ "$uninstall" -eq 1 ]; then
	if running; then
		stop_running
	fi
	remove_install
	echo "Uninstalled Catsitter. Settings were left in place."
	exit 0
fi

command -v curl >/dev/null 2>&1 || die "curl is required"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

if [ -n "$want_version" ]; then
	ver=$(strip_v "$want_version")
else
	if [ -n "$RELEASE_BASE" ]; then
		curl -fsSL "$RELEASE_BASE/latest" -o "$tmp/latest" || die "could not read the latest version"
	else
		curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" -o "$tmp/latest" || die "could not read the latest release"
	fi
	tag=$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$tmp/latest" | head -n 1)
	if [ -z "$tag" ]; then
		tag=$(tr -d ' \t\r\n' <"$tmp/latest")
	fi
	ver=$(strip_v "$tag")
fi
[ -n "$ver" ] || die "could not determine the version"
tag="v$ver"

current=$(installed_version)
if [ -n "$current" ] && [ "$current" = "$ver" ]; then
	echo "Catsitter $ver is already installed."
	exit 0
fi

case "$goos" in
linux) asset="catsitter_${ver}_linux_${goarch}.tar.gz" ;;
darwin) asset="catsitter_${ver}_darwin_${goarch}.zip" ;;
esac

if [ -n "$RELEASE_BASE" ]; then
	base="$RELEASE_BASE/download/$tag"
else
	base="https://github.com/$REPO/releases/download/$tag"
fi

echo "Downloading Catsitter $ver ($goos/$goarch)"
curl -fsSL "$base/checksums.txt" -o "$tmp/checksums.txt" || die "could not download checksums.txt"
curl -fsSL "$base/$asset" -o "$tmp/asset" || die "could not download $asset"

expected=$(awk -v name="$asset" '
	$1 ~ /^[0-9A-Fa-f]{64}$/ {
		file = $2
		sub(/^\*/, "", file)
		if (file == name) {
			print tolower($1)
			exit
		}
	}
' "$tmp/checksums.txt")
[ -n "$expected" ] || die "checksums.txt has no entry for $asset"

if command -v sha256sum >/dev/null 2>&1; then
	got=$(sha256sum "$tmp/asset" | awk '{print tolower($1)}')
elif command -v shasum >/dev/null 2>&1; then
	got=$(shasum -a 256 "$tmp/asset" | awk '{print tolower($1)}')
else
	die "sha256sum or shasum is required"
fi
[ "$got" = "$expected" ] || die "checksum mismatch for $asset"

mkdir -p "$tmp/extract"
tar -xf "$tmp/asset" -C "$tmp/extract" || die "could not extract $asset"
[ -f "$tmp/extract/catsitter" ] || die "archive did not contain catsitter"

was_running=0
if running; then
	was_running=1
	stop_running
fi

if [ "$goos" = "darwin" ]; then
	app="$HOME/Applications/Catsitter.app"
	rm -rf "$app"
	mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
	cp "$tmp/extract/catsitter" "$install_bin"
	chmod 755 "$install_bin"
	cat >"$app/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>catsitter</string>
	<key>CFBundleIdentifier</key>
	<string>com.xanderwasserman.catsitter</string>
	<key>CFBundleName</key>
	<string>Catsitter</string>
	<key>CFBundleDisplayName</key>
	<string>Catsitter</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>$ver</string>
	<key>CFBundleVersion</key>
	<string>$ver</string>
	<key>LSMinimumSystemVersion</key>
	<string>11.0</string>
	<key>LSUIElement</key>
	<true/>
	<key>NSHighResolutionCapable</key>
	<true/>
</dict>
</plist>
EOF
	if [ -f "$tmp/extract/icon.png" ]; then
		cp "$tmp/extract/icon.png" "$app/Contents/Resources/icon.png"
	elif [ -f "$tmp/extract/packaging/icon.png" ]; then
		cp "$tmp/extract/packaging/icon.png" "$app/Contents/Resources/icon.png"
	fi
	if command -v xattr >/dev/null 2>&1; then
		xattr -dr com.apple.quarantine "$app" 2>/dev/null || true
	fi
	echo "Installed Catsitter $ver to $app"
else
	mkdir -p "$HOME/.local/bin" "$HOME/.local/share/applications"
	stage="$install_bin.new"
	cp "$tmp/extract/catsitter" "$stage"
	chmod 755 "$stage"
	mv -f "$stage" "$install_bin"
	cat >"$HOME/.local/share/applications/catsitter.desktop" <<EOF
[Desktop Entry]
Type=Application
Name=Catsitter
Comment=A little cat that keeps the computer awake
Exec=$install_bin
Icon=catsitter
Terminal=false
Categories=Utility;
EOF
	icon_src=""
	if [ -f "$tmp/extract/icon.png" ]; then
		icon_src="$tmp/extract/icon.png"
	elif [ -f "$tmp/extract/packaging/icon.png" ]; then
		icon_src="$tmp/extract/packaging/icon.png"
	fi
	if [ -n "$icon_src" ]; then
		icon_dir="$HOME/.local/share/icons/hicolor/64x64/apps"
		mkdir -p "$icon_dir"
		cp "$icon_src" "$icon_dir/catsitter.png"
	fi
	echo "Installed Catsitter $ver to $install_bin"
fi

start_app
if [ "$was_running" -eq 1 ]; then
	echo "Restarted Catsitter."
else
	echo "Started Catsitter."
fi
warn_path
}

main "$@"
