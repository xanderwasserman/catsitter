//go:build darwin

package mac

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestBundleOf(t *testing.T) {
	cases := []struct {
		exe, app string
		ok       bool
	}{
		{"/Applications/Catsitter.app/Contents/MacOS/catsitter", "/Applications/Catsitter.app", true},
		{"/Users/me/Applications/Catsitter.app/Contents/MacOS/catsitter", "/Users/me/Applications/Catsitter.app", true},
		{"/Users/me/src/catsitter/catsitter", "", false},
		{"/Applications/Catsitter/Contents/MacOS/catsitter", "", false},
		{"/Applications/Catsitter.app/Contents/Resources/catsitter", "", false},
	}
	for _, c := range cases {
		app, ok := bundleOf(c.exe)
		if app != c.app || ok != c.ok {
			t.Errorf("bundleOf(%q) = %q, %v; want %q, %v", c.exe, app, ok, c.app, c.ok)
		}
	}
}

func TestRelaunchWaitsForTheOldProcess(t *testing.T) {
	old := exec.Command("/bin/sleep", "1")
	if err := old.Start(); err != nil {
		t.Fatal(err)
	}
	cmd := relaunchCommand(old.Process.Pid, "/Applications/Catsitter.app", []string{"--duration", "2h"})
	script := strings.Replace(cmd.Args[2], "/usr/bin/open", "echo", 2)
	start := time.Now()
	go old.Wait()
	out, err := exec.Command("/bin/sh", append([]string{"-c", script}, cmd.Args[3:]...)...).Output()
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("helper did not wait for the old process")
	}
	if got := strings.TrimSpace(string(out)); got != "/Applications/Catsitter.app --args --duration 2h" {
		t.Errorf("helper ran %q", got)
	}
}

func TestRelaunchWithoutArguments(t *testing.T) {
	cmd := relaunchCommand(999999, "/Applications/Catsitter.app", nil)
	script := strings.Replace(cmd.Args[2], "/usr/bin/open", "echo", 2)
	out, err := exec.Command("/bin/sh", append([]string{"-c", script}, cmd.Args[3:]...)...).Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(out)); got != "/Applications/Catsitter.app" {
		t.Errorf("helper ran %q", got)
	}
}
