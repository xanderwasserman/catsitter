//go:build linux

// Package inhibit keeps a Linux session awake.
// It tries logind, then the freedesktop screensaver, then gnome-session,
// and finally a systemd-inhibit child process.
package inhibit

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
)

// Inhibitor is the Linux sleep inhibitor.
type Inhibitor struct {
	release func() error
}

// Acquire holds one inhibitor. A second call while held does nothing.
func (i *Inhibitor) Acquire() error {
	if i.release != nil {
		return nil
	}
	var errs []error
	if rel, err := inhibitLogind(); err == nil {
		i.release = rel
		return nil
	} else {
		errs = append(errs, err)
	}
	if rel, err := inhibitScreenSaver(); err == nil {
		i.release = rel
		return nil
	} else {
		errs = append(errs, err)
	}
	if rel, err := inhibitGnome(); err == nil {
		i.release = rel
		return nil
	} else {
		errs = append(errs, err)
	}
	if rel, err := inhibitSystemd(); err == nil {
		i.release = rel
		return nil
	} else {
		errs = append(errs, err)
	}
	return fmt.Errorf("no sleep inhibitor available: %v", errs)
}

// Release drops the inhibitor.
func (i *Inhibitor) Release() error {
	if i.release == nil {
		return nil
	}
	err := i.release()
	i.release = nil
	return err
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

func inhibitGnome() (func() error, error) {
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, err
	}
	var cookie uint32
	err = conn.Object("org.gnome.SessionManager", "/org/gnome/SessionManager").Call(
		"org.gnome.SessionManager.Inhibit", 0,
		"catsitter", uint32(0), "Catsitter is keeping watch", uint32(8),
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
