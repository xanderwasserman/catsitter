//go:build windows

package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

const scriptName = "install.ps1"

const (
	createNewProcessGroup = 0x00000200
	createNoWindow        = 0x08000000
)

func scriptCommand(path, version string) *exec.Cmd {
	ps := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	if _, err := os.Stat(ps); err != nil {
		ps = "powershell.exe"
	}
	return exec.Command(ps, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path, "-Version", version)
}

// detach hides the PowerShell window and keeps Ctrl+C in this console from reaching it.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNewProcessGroup | createNoWindow}
}

// handOff starts the installer in the background and returns. Windows cannot
// replace a running catsitter.exe, so this process must exit first; the
// installer waits for that.
func handOff(cmd *exec.Cmd, dir string) error {
	log, err := os.Create(filepath.Join(dir, logName))
	if err != nil {
		return err
	}
	defer log.Close()
	cmd.Stdout = log
	cmd.Stderr = log
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	fmt.Println("Installing in the background. Catsitter restarts in a few seconds.")
	return cmd.Process.Release()
}
