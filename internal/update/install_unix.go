//go:build !windows

package update

import (
	"os"
	"os/exec"
	"syscall"
)

const scriptName = "install.sh"

func scriptCommand(path, version string) *exec.Cmd {
	return exec.Command("/bin/sh", path, "--version", version)
}

// detach starts a new session, so the installer's pkill and the end of this
// process leave it running.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// handOff replaces this process with the installer. The terminal sees the
// installer's output, and the installer's pkill cannot stop it halfway.
func handOff(cmd *exec.Cmd, _ string) error {
	if err := os.Chdir(cmd.Dir); err != nil {
		return err
	}
	return syscall.Exec(cmd.Path, cmd.Args, cmd.Env)
}
