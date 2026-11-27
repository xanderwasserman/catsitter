//go:build darwin

package mac

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// relaunchScript waits up to 30 seconds for the old process to exit, then
// opens the app through Launch Services with the same arguments. If that
// fails and no copy of the app runs for this user, it runs the app's
// executable directly.
const relaunchScript = `pid=$1; app=$2; exe=$3; shift 3
tries=150
while kill -0 "$pid" 2>/dev/null; do
	tries=$((tries - 1))
	[ "$tries" -gt 0 ] || exit 1
	sleep 0.2
done
if [ "$#" -gt 0 ]; then /usr/bin/open "$app" --args "$@"; else /usr/bin/open "$app"; fi ||
	/usr/bin/pgrep -x -U "$(id -u)" "${exe##*/}" >/dev/null || exec "$exe" "$@"`

// relaunchMarkerAge is how long after Relaunch the new copy still counts as
// relaunched.
const relaunchMarkerAge = time.Minute

// Relauncher starts this app bundle again once the current process exits.
// Marker is the file that tells the new copy which process relaunched it.
type Relauncher struct {
	Marker string
}

// Relaunch starts a detached helper that opens this app bundle once the
// current process has exited. It reports false for a binary outside a bundle.
func (r Relauncher) Relaunch() bool {
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
	if err := r.writeMarker(); err != nil {
		log.Printf("relaunch: %v", err)
		return false
	}
	cmd := relaunchCommand(os.Getpid(), app, exe, os.Args[1:])
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		os.Remove(r.Marker)
		log.Printf("relaunch: %v", err)
		return false
	}
	cmd.Process.Release()
	return true
}

// Relaunched reports whether Relaunch started this process. It reads the
// marker once and removes it.
func (r Relauncher) Relaunched() bool {
	info, err := os.Stat(r.Marker)
	if err != nil {
		return false
	}
	data, err := os.ReadFile(r.Marker)
	os.Remove(r.Marker)
	if err != nil {
		return false
	}
	return fromRelaunch(data, time.Since(info.ModTime()), os.Getpid(), processAlive)
}

func (r Relauncher) writeMarker() error {
	if err := os.MkdirAll(filepath.Dir(r.Marker), 0o700); err != nil {
		return err
	}
	return os.WriteFile(r.Marker, []byte(strconv.Itoa(os.Getpid())), 0o600)
}

// fromRelaunch reports whether a marker of the given age, holding the pid of
// the process that wrote it, belongs to a relaunch of the process self.
func fromRelaunch(data []byte, age time.Duration, self int, alive func(int) bool) bool {
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 || pid == self {
		return false
	}
	return age >= 0 && age < relaunchMarkerAge && !alive(pid)
}

func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
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

func relaunchCommand(pid int, app, exe string, args []string) *exec.Cmd {
	argv := append([]string{"-c", relaunchScript, "catsitter-relaunch", strconv.Itoa(pid), app, exe}, args...)
	return exec.Command("/bin/sh", argv...)
}
