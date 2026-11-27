//go:build darwin

package mac

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// relaunchScript waits for the old process to exit, then opens the app
// through Launch Services with the same arguments.
const relaunchScript = `pid=$1; app=$2; shift 2
while kill -0 "$pid" 2>/dev/null; do sleep 0.2; done
if [ "$#" -gt 0 ]; then exec /usr/bin/open "$app" --args "$@"; fi
exec /usr/bin/open "$app"`

// relaunchMarkerAge is how long after Relaunch the new copy still counts as
// relaunched.
const relaunchMarkerAge = time.Minute

// Relaunch starts a detached helper that opens this app bundle once the
// current process has exited. It reports false for a binary outside a bundle.
func Relaunch() bool {
	exe, err := os.Executable()
	if err != nil {
		log.Printf("relaunch: %v", err)
		return false
	}
	app, ok := bundleOf(exe)
	if !ok {
		log.Printf("relaunch: %s is not inside an app bundle", exe)
		return false
	}
	marker := relaunchMarker()
	if err := os.WriteFile(marker, nil, 0o600); err != nil {
		log.Printf("relaunch: %v", err)
		return false
	}
	cmd := relaunchCommand(os.Getpid(), app, os.Args[1:])
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		os.Remove(marker)
		log.Printf("relaunch: %v", err)
		return false
	}
	cmd.Process.Release()
	return true
}

// Relaunched reports whether Relaunch started this process. It answers true
// once per relaunch.
func Relaunched() bool {
	marker := relaunchMarker()
	info, err := os.Stat(marker)
	if err != nil {
		return false
	}
	os.Remove(marker)
	return time.Since(info.ModTime()) < relaunchMarkerAge
}

func relaunchMarker() string {
	return filepath.Join(os.TempDir(), "catsitter-relaunched")
}

// bundleOf returns the .app bundle that holds exe in Contents/MacOS.
func bundleOf(exe string) (string, bool) {
	macOS := filepath.Dir(exe)
	contents := filepath.Dir(macOS)
	app := filepath.Dir(contents)
	if filepath.Base(macOS) != "MacOS" || filepath.Base(contents) != "Contents" || !strings.HasSuffix(app, ".app") {
		return "", false
	}
	return app, true
}

func relaunchCommand(pid int, app string, args []string) *exec.Cmd {
	argv := append([]string{"-c", relaunchScript, "catsitter-relaunch", strconv.Itoa(pid), app}, args...)
	return exec.Command("/bin/sh", argv...)
}
