#!/bin/sh
# Install Catsitter for the current user. No sudo.
#
#   curl -fsSL https://github.com/xanderwasserman/catsitter/releases/latest/download/install.sh | sh
#   curl -fsSL .../install.sh | sh -s -- --version 0.1.0
#   curl -fsSL .../install.sh | sh -s -- --uninstall
set -eu

REPO="${CATSITTER_REPO:-xanderwasserman/catsitter}"
RELEASE_BASE="${CATSITTER_RELEASE_BASE:-}"
# A running Catsitter hands over a verified download in this directory. It is
# not passed on to the Catsitter this script starts.
if [ -n "${CATSITTER_INSTALL_FROM:-}" ]; then
	RELEASE_BASE=$CATSITTER_INSTALL_FROM
fi
unset CATSITTER_INSTALL_FROM

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

bundle_id=com.xanderwasserman.catsitter
app="$HOME/Applications/Catsitter.app"
if [ "$goos" = "darwin" ]; then
	install_bin="$app/Contents/MacOS/catsitter"
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
	# Through Launch Services, macOS checks Accessibility for Catsitter itself
	# rather than for this terminal.
	if [ "$goos" = "darwin" ] && open "$app" >/dev/null 2>&1; then
		return 0
	fi
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

# The Accessibility entry is keyed by the bundle identifier and pinned to the
# exact build it was granted to. A new build would leave it looking on while
# macOS refuses it, so it is cleared and Catsitter asks again.
forget_permission() {
	if command -v tccutil >/dev/null 2>&1; then
		tccutil reset Accessibility "$bundle_id" >/dev/null 2>&1 || true
	fi
}

# signed_as_catsitter reports whether a bundle's code identity is the bundle
# identifier. Copies installed by Catsitter 0.1.7 or earlier are not.
signed_as_catsitter() {
	codesign -dv "$1" 2>&1 | grep -qx "Identifier=$bundle_id"
}

# build_bundle assembles and signs Catsitter.app at $1 from the extracted
# release. An ad-hoc signature over the whole bundle makes the bundle
# identifier the code identity that Accessibility records.
build_bundle() {
	mkdir -p "$1/Contents/MacOS" "$1/Contents/Resources"
	cp "$tmp/extract/catsitter" "$1/Contents/MacOS/catsitter"
	chmod 755 "$1/Contents/MacOS/catsitter"
	cat >"$1/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>catsitter</string>
	<key>CFBundleIconFile</key>
	<string>icon</string>
	<key>CFBundleIdentifier</key>
	<string>$bundle_id</string>
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
	if [ -f "$tmp/extract/icon.icns" ]; then
		cp "$tmp/extract/icon.icns" "$1/Contents/Resources/icon.icns"
	fi
	xattr -cr "$1"
	codesign --force --sign - "$1" || die "could not sign Catsitter.app"
	codesign --verify --strict "$1" || die "Catsitter.app failed signature verification"
}

# place_bundle swaps the staged bundle at $1 into place. If that fails, the
# previous app is put back and started again if it was running.
place_bundle() {
	old="$app.old"
	rm -rf "$old"
	mkdir -p "$HOME/Applications"
	if [ -d "$app" ]; then
		mv "$app" "$old" || restore_and_die "could not move the previous Catsitter.app aside"
	fi
	if ! mv "$1" "$app"; then
		rm -rf "$app"
		restore_and_die "could not move Catsitter.app into place"
	fi
	rm -rf "$old"
}

restore_and_die() {
	if [ -d "$old" ] && [ ! -e "$app" ]; then
		mv "$old" "$app" || true
	fi
	if [ "$was_running" -eq 1 ] && [ -d "$app" ]; then
		start_app
	fi
	die "$1"
}

remove_install() {
	if [ "$goos" = "darwin" ]; then
		rm -rf "$app"
		forget_permission
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

if [ "$goos" = "darwin" ]; then
	for tool in codesign xattr; do
		command -v "$tool" >/dev/null 2>&1 || die "$tool is required"
	done
fi

# fetch copies a release file from a URL or a local directory.
fetch() {
	case "$1" in
	http://* | https://* | file://*)
		command -v curl >/dev/null 2>&1 || die "curl is required"
		curl -fsSL "$1" -o "$2"
		;;
	*)
		cp "$1" "$2"
		;;
	esac
}

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT INT TERM

if [ -n "$want_version" ]; then
	ver=$(strip_v "$want_version")
else
	if [ -n "$RELEASE_BASE" ]; then
		fetch "$RELEASE_BASE/latest" "$tmp/latest" || die "could not read the latest version"
	else
		fetch "https://api.github.com/repos/$REPO/releases/latest" "$tmp/latest" || die "could not read the latest release"
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
fetch "$base/checksums.txt" "$tmp/checksums.txt" || die "could not download checksums.txt"
fetch "$base/$asset" "$tmp/asset" || die "could not download $asset"

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

if [ "$goos" = "darwin" ]; then
	build_bundle "$tmp/Catsitter.app"
	legacy=0
	if [ -d "$app" ] && ! signed_as_catsitter "$app"; then
		legacy=1
	fi
fi

was_running=0
if running; then
	was_running=1
	stop_running
fi

if [ "$goos" = "darwin" ]; then
	updating=0
	if [ -d "$app" ]; then
		updating=1
	fi
	place_bundle "$tmp/Catsitter.app"
	forget_permission
	# Re-register the bundle so Finder, Spotlight and Launchpad show the new icon.
	touch "$app"
	lsregister=/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister
	if [ -x "$lsregister" ]; then
		"$lsregister" -f "$app" >/dev/null 2>&1 || true
	fi
	if [ "$updating" -eq 1 ]; then
		echo "After this update, turn Catsitter on again in Privacy & Security > Accessibility. macOS ties the permission to each build."
	fi
	if [ "$legacy" -eq 1 ]; then
		echo "Catsitter 0.1.7 and earlier may have left Accessibility entries that no longer work. Select each Catsitter there and click - to remove it before turning on the new one."
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
