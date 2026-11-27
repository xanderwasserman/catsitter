//go:build darwin

package mac

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

const testApp = "/Applications/Catsitter.app"

func TestBundleOf(t *testing.T) {
	cases := []struct {
		exe, app string
		ok       bool
	}{
		{testApp + "/Contents/MacOS/catsitter", testApp, true},
		{"/Users/me/Applications/Catsitter.app/Contents/MacOS/catsitter", "/Users/me/Applications/Catsitter.app", true},
		{"/Users/me/src/catsitter/catsitter", "", false},
		{"/Applications/Catsitter/Contents/MacOS/catsitter", "", false},
		{testApp + "/Contents/Resources/catsitter", "", false},
	}
	for _, c := range cases {
		app, ok := bundleOf(c.exe)
		if app != c.app || ok != c.ok {
			t.Errorf("bundleOf(%q) = %q, %v; want %q, %v", c.exe, app, ok, c.app, c.ok)
		}
	}
}

// runHelper runs the relaunch helper with open replaced by openCmd.
func runHelper(t *testing.T, pid int, exe, openCmd string, args ...string) (string, error) {
	t.Helper()
	cmd := relaunchCommand(pid, testApp, exe, args)
	script := strings.ReplaceAll(cmd.Args[2], "/usr/bin/open", openCmd)
	out, err := exec.Command("/bin/sh", append([]string{"-c", script}, cmd.Args[3:]...)...).Output()
	return strings.TrimSpace(string(out)), err
}

func TestRelaunchWaitsForTheOldProcess(t *testing.T) {
	old := exec.Command("/bin/sleep", "1")
	if err := old.Start(); err != nil {
		t.Fatal(err)
	}
	go old.Wait()
	start := time.Now()
	out, err := runHelper(t, old.Process.Pid, "/bin/false", "echo", "--duration", "2h")
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("helper did not wait for the old process")
	}
	if out != testApp+" --args --duration 2h" {
		t.Errorf("helper ran %q", out)
	}
}

func TestRelaunchWithoutArguments(t *testing.T) {
	out, err := runHelper(t, 999999, "/bin/false", "echo")
	if err != nil {
		t.Fatal(err)
	}
	if out != testApp {
		t.Errorf("helper ran %q", out)
	}
}

func TestRelaunchRunsTheExecutableWhenOpenFails(t *testing.T) {
	out, err := runHelper(t, 999999, "/bin/echo", "false", "--duration", "2h")
	if err != nil {
		t.Fatal(err)
	}
	if out != "--duration 2h" {
		t.Errorf("fallback ran with %q", out)
	}
}

func TestFromRelaunch(t *testing.T) {
	dead := func(int) bool { return false }
	alive := func(int) bool { return true }
	cases := []struct {
		name  string
		data  string
		age   time.Duration
		self  int
		alive func(int) bool
		want  bool
	}{
		{"fresh marker from an exited process", "100", time.Second, 200, dead, true},
		{"old process still running", "100", time.Second, 200, alive, false},
		{"stale marker", "100", 2 * time.Minute, 200, dead, false},
		{"own pid", "200", time.Second, 200, dead, false},
		{"not a pid", "catsitter", time.Second, 200, dead, false},
		{"empty", "", time.Second, 200, dead, false},
	}
	for _, c := range cases {
		if got := fromRelaunch([]byte(c.data), c.age, c.self, c.alive); got != c.want {
			t.Errorf("%s: got %v", c.name, got)
		}
	}
}

func TestRelaunchedReadsTheMarkerOnce(t *testing.T) {
	r := Relauncher{Marker: filepath.Join(t.TempDir(), "relaunch")}
	if err := os.WriteFile(r.Marker, []byte(strconv.Itoa(999999)), 0o600); err != nil {
		t.Fatal(err)
	}
	if !r.Relaunched() {
		t.Fatal("marker from an exited process was not recognised")
	}
	if r.Relaunched() {
		t.Fatal("marker was recognised twice")
	}
}
