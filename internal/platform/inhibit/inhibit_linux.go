//go:build linux

// Package inhibit keeps a Linux session awake.
// It tries logind, then the freedesktop screensaver, then gnome-session,
// and finally a systemd-inhibit child process.
package inhibit

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
)

// Inhibitor is the Linux sleep inhibitor.
// It holds system sleep and, when a desktop offers it, screen blank and lock.
type Inhibitor struct {
	releases []func() error
}

// Acquire holds sleep and screen inhibitors. A second call while held does nothing.
func (i *Inhibitor) Acquire() error {
	if len(i.releases) > 0 {
		return nil
	}
	var held []func() error
	var notes []string
	haveSleep := false
	if rel, err := inhibitLogind(); err == nil {
		held = append(held, rel)
		haveSleep = true
	} else {
		notes = append(notes, "logind: "+err.Error())
		if rel, err := inhibitSystemd(); err == nil {
			held = append(held, rel)
			haveSleep = true
		} else {
			notes = append(notes, "systemd-inhibit: "+err.Error())
		}
	}

	haveScreen := false
	gnomeFlags := uint32(8) // inhibit session idle
	if !haveSleep {
		gnomeFlags |= 4 // inhibit suspend
	}
	if rel, err := inhibitScreenSaver(); err == nil {
		held = append(held, rel)
		haveScreen = true
	} else {
		notes = append(notes, "screensaver: "+err.Error())
		if rel, err := inhibitGnome(gnomeFlags); err == nil {
			held = append(held, rel)
			haveScreen = true
		} else {
			notes = append(notes, "gnome session: "+err.Error())
		}
	}
	if len(held) == 0 {
		return fmt.Errorf("no sleep inhibitor available (%s)", strings.Join(notes, "; "))
	}
	if !haveSleep {
		log.Printf("system sleep is not inhibited (%s)", strings.Join(notes, "; "))
	}
	if !haveScreen {
		log.Printf("the screen can still lock or blank (%s)", strings.Join(notes, "; "))
	}
	i.releases = held
	return nil
}

// Release drops every inhibitor that was acquired.
func (i *Inhibitor) Release() error {
	left, err := dropReleased(i.releases)
	i.releases = left
	return err
}

func dropReleased(rels []func() error) (left []func() error, err error) {
	for _, rel := range rels {
		if e := rel(); e != nil {
			if err == nil {
				err = e
			}
			left = append(left, rel)
		}
	}
	return left, err
}

func inhibitLogind() (func() error, error) {
	conn, err := dbus.SystemBus()
	if err != nil {
		return nil, err
	}
	call := conn.Object("org.freedesktop.login1", "/org/freedesktop/login1").Call(
		"org.freedesktop.login1.Manager.Inhibit", 0,
		"idle:sleep", "catsitter", "Catsitter is keeping watch", "block",
	)
	if call.Err != nil {
		return nil, call.Err
	}
	var fd dbus.UnixFD
	if err := call.Store(&fd); err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), "catsitter-inhibit")
	return func() error { return file.Close() }, nil
}

func inhibitScreenSaver() (func() error, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	names := []struct{ dest, path string }{
		{"org.freedesktop.ScreenSaver", "/org/freedesktop/ScreenSaver"},
		{"org.freedesktop.ScreenSaver", "/ScreenSaver"},
		{"org.gnome.ScreenSaver", "/org/gnome/ScreenSaver"},
	}
	var first error
	for _, n := range names {
		var cookie uint32
		err := conn.Object(n.dest, dbus.ObjectPath(n.path)).Call(
			n.dest+".Inhibit", 0, "catsitter", "Catsitter is keeping watch",
		).Store(&cookie)
		if err != nil {
			if first == nil {
				first = err
			}
			continue
		}
		dest, path := n.dest, n.path
		return func() error {
			return conn.Object(dest, dbus.ObjectPath(path)).Call(dest+".UnInhibit", 0, cookie).Err
		}, nil
	}
	if first == nil {
		first = fmt.Errorf("screensaver inhibit unavailable")
	}
	return nil, first
}

func inhibitGnome(flags uint32) (func() error, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	var cookie uint32
	err = conn.Object("org.gnome.SessionManager", "/org/gnome/SessionManager").Call(
		"org.gnome.SessionManager.Inhibit", 0,
		"catsitter", uint32(0), "Catsitter is keeping watch", flags,
	).Store(&cookie)
	if err != nil {
		return nil, err
	}
	return func() error {
		return conn.Object("org.gnome.SessionManager", "/org/gnome/SessionManager").Call(
			"org.gnome.SessionManager.Uninhibit", 0, cookie,
		).Err
	}, nil
}

func inhibitSystemd() (func() error, error) {
	cmd := exec.Command("systemd-inhibit",
		"--what=idle:sleep",
		"--who=catsitter",
		"--why=Catsitter is keeping watch",
		"--mode=block",
		"sleep", "99999999",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err == nil {
			return nil, fmt.Errorf("systemd-inhibit exited")
		}
		return nil, err
	case <-time.After(200 * time.Millisecond):
	}
	return func() error {
		if cmd.Process == nil {
			return nil
		}
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			<-done
		}
		return nil
	}, nil
}
