#!/bin/sh
# One-off check of the macOS Accessibility flow on a CI runner. Each check
# prints PASS or FAIL, screenshots go to SHOTS, and it exits 1 if any failed.
#
#   sh packaging/test-app.sh BINARY SHOTS
set -u

bin=$1
shots=$2
root=$(cd "$(dirname "$0")/.." && pwd)
id=com.xanderwasserman.catsitter
db="/Library/Application Support/com.apple.TCC/TCC.db"
checker="$(xcode-select -p)/usr/lib/libMainThreadChecker.dylib"
work=$(mktemp -d)
log="$work/log"
failed=0
[ "${CI:-}" = "true" ] || { echo "CI only: this edits the TCC database" >&2; exit 1; }
mkdir -p "$shots"
save_logs() {
	cp "$log" "$shots/app-stderr.log" 2>/dev/null || true
	cp "$work/out" "$shots/app-stdout.log" 2>/dev/null || true
	sudo sqlite3 "$db" "select service,client,auth_value,auth_reason from access where client='$id'" >"$shots/tcc-rows.txt" 2>&1 || true
	log show --last 10m --style compact --predicate 'process == "catsitter" OR (subsystem == "com.apple.TCC" AND eventMessage CONTAINS "catsitter")' >"$shots/unified.log" 2>&1 || true
	sh "$root/install.sh" --uninstall >/dev/null 2>&1 || true
	rm -rf "$work"
}
trap save_logs EXIT
clang -framework ApplicationServices -o "$work/click" "$root/packaging/click.m" || echo "FAIL build the click helper"

check() {
	if eval "$2" >/dev/null 2>&1; then echo "PASS $1"; else echo "FAIL $1"; failed=1; fi
}
wait_for() {
	i=0
	until eval "$1" >/dev/null 2>&1; do
		i=$((i + 1))
		[ "$i" -lt 60 ] || return 1
		sleep 0.5
	done
}
shot() { screencapture -x "$shots/$1.png" || true; }
row() { sudo sqlite3 "$db" "select auth_value from access where service='kTCCServiceAccessibility' and client='$id'"; }
on_screen() {
	osascript -e "tell application \"System Events\" to tell process \"$1\" to get entire contents of every window" 2>/dev/null | grep -q "$2"
}
click_item() {
	pos=$(osascript -e "tell application \"System Events\" to tell process \"catsitter\" to get {position, size} of $1" 2>/dev/null || true)
	set -- $(echo "$pos" | tr -d ',') "$2"
	[ $# -eq 5 ] || return 0
	if [ "$5" = right ]; then
		"$work/click" $(($1 + $3 / 2)) $(($2 + $4 / 2)) right
	else
		"$work/click" $(($1 + $3 / 2)) $(($2 + $4 / 2))
	fi
	sleep 1.5
}
escape() { osascript -e 'tell application "System Events" to key code 53' >/dev/null 2>&1 || true; sleep 1; }

ver=0.0.0-ci
release="$work/release/download/v$ver"
mkdir -p "$release" "$work/stage"
cp "$bin" "$work/stage/catsitter"
cp "$root/packaging/icon.icns" "$work/stage/icon.icns"
(cd "$work/stage" && zip -q "$release/catsitter_${ver}_darwin_arm64.zip" catsitter icon.icns)
(cd "$release" && shasum -a 256 ./*.zip | sed 's| \./| |' >checksums.txt)

sudo sqlite3 "$db" "delete from access where client='$id'"
CATSITTER_RELEASE_BASE="file://$work/release" sh "$root/install.sh" --version "$ver"
app="$HOME/Applications/Catsitter.app"
wait_for "pgrep -x catsitter"
sleep 4
shot 1-first-launch
check "the cat window is on screen" 'osascript -e "tell application \"System Events\" to tell process \"catsitter\" to get window \"Catsitter\""'
check "first launch adds Catsitter to the TCC list, switched off" '[ "$(row)" = 0 ]'
check "explanation panel is on screen after install" 'on_screen catsitter "Let Catsitter nudge your Mac"'
pkill -x catsitter
wait_for "! pgrep -x catsitter"
check "Catsitter quits on SIGTERM while the panel is open" '! pgrep -x catsitter'

open --env DYLD_INSERT_LIBRARIES="$checker" --env MTC_CRASH_ON_REPORT=1 --stdout "$work/out" --stderr "$log" "$app"
wait_for "grep -q 'accessibility permission is missing' '$log'"
wait_for "on_screen catsitter 'Let Catsitter nudge your Mac'"
shot 2-panel
check "panel shows under the Main Thread Checker" 'on_screen catsitter "Let Catsitter nudge your Mac"'
check "panel window title comes from Go" 'on_screen catsitter "Allow Accessibility"'
sleep 10
check "still running 10s later without permission" "pgrep -x catsitter"
check "stays out of the Dock and Cmd+Tab" 'lsappinfo info -only ApplicationType "$(lsappinfo find pid="$(pgrep -x catsitter)")" | grep -q UIElement'

click_item 'window "Catsitter"' right
shot 3-cat-menu-untrusted
escape
click_item 'menu bar item 1 of menu bar 2' left
shot 4-tray-untrusted-light
escape

osascript -e 'tell application "System Events" to tell process "catsitter" to click button "Open Settings" of window "Allow Accessibility"' >/dev/null 2>&1
wait_for "pgrep -x 'System Settings'"
sleep 3
shot 5-open-settings
check "Open Settings opens System Settings" "pgrep -x 'System Settings'"
wait_for "on_screen 'System Settings' 'Catsitter'"
shot 6-settings-list
check "Catsitter is in the Accessibility list" "on_screen 'System Settings' 'Catsitter'"
osascript -e 'tell application "System Events" to get name of every process whose visible is true' 2>&1 | tr ',' '\n' | sed 's/^ */  visible: /'
echo "TCC rows for Catsitter:"
sudo sqlite3 "$db" "select service,auth_value,auth_reason from access where client='$id'"

sudo sqlite3 "$db" "update access set auth_value=2 where service='kTCCServiceAccessibility' and client='$id'"
sudo killall tccd || true
wait_for "grep -q 'accessibility permission granted' '$log'"
check "grant is noticed without a restart" "grep -q 'accessibility permission granted' '$log'"
wait_for "on_screen catsitter 'Catsitter can nudge now'"
shot 7-granted
check "thanks note is on screen" "on_screen catsitter 'Catsitter can nudge now'"
check "panel closed after the grant" '! on_screen catsitter "Let Catsitter nudge your Mac"'
sleep 5
click_item 'menu bar item 1 of menu bar 2' left
shot 8-tray-trusted-light
escape
osascript -e 'tell application "System Events" to tell appearance preferences to set dark mode to true' >/dev/null 2>&1 || true
sleep 2
click_item 'menu bar item 1 of menu bar 2' left
shot 9-tray-trusted-dark
escape
click_item 'window "Catsitter"' right
shot 10-cat-menu-trusted
escape

check "Catsitter is still running" "pgrep -x catsitter"
check "no Main Thread Checker reports" "! grep -q 'Main Thread Checker' '$log'"
echo "--- app log"
cat "$log"
exit "$failed"
