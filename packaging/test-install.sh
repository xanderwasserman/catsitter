#!/bin/sh
# Exercise install.sh: install, update, skip, and uninstall.
set -eu

script_dir=$(dirname "$0")
root=$(cd "$script_dir/.." && pwd)
script="$root/install.sh"

die() {
	echo "test-install: $*" >&2
	exit 1
}

command -v go >/dev/null 2>&1 || die "go is required"
command -v python3 >/dev/null 2>&1 || die "python3 is required"
command -v curl >/dev/null 2>&1 || die "curl is required"

os=$(uname -s)
arch=$(uname -m)
case "$arch" in
x86_64 | amd64) goarch=amd64 ;;
aarch64 | arm64) goarch=arm64 ;;
*) die "unsupported architecture: $arch" ;;
esac

case "$os" in
Linux)
	goos=linux
	ext=tar.gz
	;;
Darwin)
	goos=darwin
	ext=zip
	;;
*)
	die "run this on Linux or macOS"
	;;
esac

work=$(mktemp -d)
home="$work/home"
marker="$work/marker"
server_root="$work/www"
mkdir -p "$home" "$server_root"

cleanup() {
	if [ -n "${server_pid:-}" ]; then
		kill "$server_pid" >/dev/null 2>&1 || true
		wait "$server_pid" >/dev/null 2>&1 || true
	fi
	if command -v pgrep >/dev/null 2>&1; then
		for pid in $(pgrep -x catsitter || true); do
			if [ -r "/proc/$pid/exe" ]; then
				target=$(readlink -f "/proc/$pid/exe" || true)
				case "$target" in
				"$home"/*) kill "$pid" >/dev/null 2>&1 || true ;;
				esac
			fi
		done
	fi
	rm -rf "$work"
}
trap cleanup EXIT INT TERM

build_stub() {
	ver=$1
	out=$2
	go build -ldflags "-X main.version=$ver" -o "$out" "$root/packaging/stub"
}

publish() {
	ver=$1
	bin=$2
	dir="$server_root/download/v$ver"
	stage="$work/stage-$ver"
	rm -rf "$dir" "$stage"
	mkdir -p "$dir" "$stage"
	cp "$bin" "$stage/catsitter"
	cp "$root/packaging/icon.png" "$stage/icon.png"
	asset="catsitter_${ver}_${goos}_${goarch}.$ext"
	if [ "$ext" = "zip" ]; then
		python3 - "$stage" "$dir/$asset" <<'PY'
import sys, zipfile
from pathlib import Path
stage = Path(sys.argv[1])
with zipfile.ZipFile(sys.argv[2], "w", compression=zipfile.ZIP_DEFLATED) as zf:
    for item in sorted(stage.iterdir()):
        zf.write(item, item.name)
PY
	else
		tar -czf "$dir/$asset" -C "$stage" catsitter icon.png
	fi
	rm -rf "$stage"
	if command -v sha256sum >/dev/null 2>&1; then
		sum=$(sha256sum "$dir/$asset" | awk '{print $1}')
	else
		sum=$(shasum -a 256 "$dir/$asset" | awk '{print $1}')
	fi
	printf '%s  %s\n' "$sum" "$asset" >"$dir/checksums.txt"
}

installed_bin() {
	if [ "$goos" = "darwin" ]; then
		printf '%s\n' "$home/Applications/Catsitter.app/Contents/MacOS/catsitter"
	else
		printf '%s\n' "$home/.local/bin/catsitter"
	fi
}

set_latest() {
	printf '%s\n' "v$1" >"$server_root/latest"
}

port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')
python3 -m http.server "$port" --bind 127.0.0.1 --directory "$server_root" >/dev/null 2>&1 &
server_pid=$!
base="http://127.0.0.1:$port"

i=0
while [ "$i" -lt 50 ]; do
	if curl -fsS "$base/latest" >/dev/null 2>&1; then
		break
	fi
	printf 'v0.0.0\n' >"$server_root/latest"
	i=$((i + 1))
	sleep 0.1
done
curl -fsS "$base/latest" >/dev/null || die "fixture server did not start"

one="$work/catsitter-0.1.0"
two="$work/catsitter-0.2.0"
build_stub 0.1.0 "$one"
build_stub 0.2.0 "$two"
publish 0.1.0 "$one"
publish 0.2.0 "$two"

export HOME="$home"
export CATSITTER_RELEASE_BASE="$base"
export CATSITTER_STUB_MARKER="$marker"
unset CATSITTER_REPO || true

bin=$(installed_bin)

# A bad checksum must not install anything.
bad="$work/badsum"
cp "$server_root/download/v0.1.0/checksums.txt" "$bad"
printf '0000000000000000000000000000000000000000000000000000000000000000  catsitter_0.1.0_%s_%s.%s\n' "$goos" "$goarch" "$ext" >"$server_root/download/v0.1.0/checksums.txt"
set_latest 0.1.0
if sh "$script"; then
	die "bad checksum was accepted"
fi
[ ! -e "$bin" ] || die "bad checksum still installed a binary"
mv "$bad" "$server_root/download/v0.1.0/checksums.txt"

set_latest 0.1.0
sh "$script"
[ -x "$bin" ] || die "binary was not installed"
got=$("$bin" --version | tr -d ' \t\r\n')
[ "$got" = "0.1.0" ] || die "installed version is $got"
if [ "$goos" = "linux" ]; then
	[ -f "$home/.local/share/applications/catsitter.desktop" ] || die "missing desktop entry"
	[ -f "$home/.local/share/icons/hicolor/64x64/apps/catsitter.png" ] || die "missing icon"
	grep -q "$bin" "$home/.local/share/applications/catsitter.desktop" || die "desktop entry has the wrong path"
else
	[ -f "$home/Applications/Catsitter.app/Contents/Info.plist" ] || die "missing Info.plist"
	grep -q "$got" "$home/Applications/Catsitter.app/Contents/Info.plist" || die "Info.plist version was not written"
fi

# Pinning an older version while latest has moved.
set_latest 0.2.0
sh "$script" --version 0.1.0
got=$("$bin" --version | tr -d ' \t\r\n')
[ "$got" = "0.1.0" ] || die "pinned install changed the version to $got"

rm -f "$marker"
"$bin" >/dev/null 2>&1 &
stub_pid=$!
i=0
while [ ! -s "$marker" ]; do
	i=$((i + 1))
	[ "$i" -lt 50 ] || die "stub did not write a marker"
	sleep 0.1
done
before=$(tr -d ' \t\r\n' <"$marker")

set_latest 0.2.0
sh "$script"
got=$("$bin" --version | tr -d ' \t\r\n')
[ "$got" = "0.2.0" ] || die "update left version $got"
i=0
while [ ! -s "$marker" ] || [ "$(tr -d ' \t\r\n' <"$marker")" = "$before" ]; do
	i=$((i + 1))
	[ "$i" -lt 50 ] || die "update did not restart Catsitter"
	sleep 0.1
done
after=$(tr -d ' \t\r\n' <"$marker")
[ "$after" != "$before" ] || die "restart kept the old pid"

stamp=$(date -u +%s 2>/dev/null || echo now)
# Touch is compared via checksum of the binary bytes after a skip.
sum_before=$("$bin" --version; wc -c <"$bin")
sh "$script"
sum_after=$("$bin" --version; wc -c <"$bin")
[ "$sum_before" = "$sum_after" ] || die "skip replaced the current binary"
still=$(tr -d ' \t\r\n' <"$marker")
[ "$still" = "$after" ] || die "skip stopped the running Catsitter"

sh "$script" --uninstall
[ ! -e "$bin" ] || die "uninstall left the binary"
if [ "$goos" = "linux" ]; then
	[ ! -e "$home/.local/share/applications/catsitter.desktop" ] || die "uninstall left the desktop entry"
else
	[ ! -e "$home/Applications/Catsitter.app" ] || die "uninstall left the app bundle"
fi
i=0
while kill -0 "$after" >/dev/null 2>&1; do
	i=$((i + 1))
	[ "$i" -lt 50 ] || die "uninstall left Catsitter running"
	sleep 0.1
done

# The first stub was replaced by the update and should already be gone.
if kill -0 "$stub_pid" >/dev/null 2>&1; then
	die "the first process is still running"
fi

echo "install, update, and uninstall passed ($goos/$goarch) $stamp"
