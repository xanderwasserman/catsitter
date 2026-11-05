#!/bin/sh
# Launch a macOS build under the Main Thread Checker and check that it keeps running.
#
#   sh packaging/test-launch.sh BINARY [SECONDS]
set -eu

die() {
	echo "test-launch: $*" >&2
	exit 1
}

[ $# -ge 1 ] || die "usage: test-launch.sh BINARY [SECONDS]"
bin=$1
seconds=${2:-15}
[ -x "$bin" ] || die "$bin is not executable"

checker="$(xcode-select -p)/usr/lib/libMainThreadChecker.dylib"
[ -f "$checker" ] || die "missing $checker"

work=$(mktemp -d)
log="$work/log"
pid=""
cleanup() {
	if [ -n "$pid" ]; then
		kill "$pid" >/dev/null 2>&1 || true
		wait "$pid" >/dev/null 2>&1 || true
	fi
	rm -rf "$work"
}
trap cleanup EXIT INT TERM

HOME="$work" DYLD_INSERT_LIBRARIES="$checker" MTC_CRASH_ON_REPORT=1 "$bin" >"$log" 2>&1 &
pid=$!
sleep "$seconds"

alive=1
kill -0 "$pid" >/dev/null 2>&1 || alive=0
kind=$(lsappinfo info -only ApplicationType "$(lsappinfo find "pid=$pid")" 2>/dev/null || true)
cat "$log"
if grep -q "Main Thread Checker" "$log"; then
	die "UI API called off the main thread"
fi
[ "$alive" -eq 1 ] || die "Catsitter exited within ${seconds}s"
case "$kind" in
*UIElement*) ;;
*) die "Catsitter is not a UI element, so the Dock and Cmd+Tab show it: $kind" ;;
esac
echo "Catsitter kept running for ${seconds}s as a UI element with no main-thread reports"
