package update

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Installer runs the install script that ships inside the binary against a
// staged release. The script stops the running Catsitter, replaces it and
// starts the new version, exactly as a manual re-install does.
type Installer struct {
	// Script is install.sh on macOS and Linux, install.ps1 on Windows.
	Script []byte
}

const logName = "install.log"

func (in Installer) command(dir string, v Version) (*exec.Cmd, error) {
	if len(in.Script) == 0 {
		return nil, errors.New("this build has no installer")
	}
	path := filepath.Join(dir, scriptName)
	if err := os.WriteFile(path, in.Script, 0o700); err != nil {
		return nil, err
	}
	cmd := scriptCommand(path, v.String())
	cmd.Dir = filepath.Dir(dir)
	cmd.Env = append(os.Environ(), "CATSITTER_INSTALL_FROM="+dir)
	return cmd, nil
}

// Start runs the installer in the background, detached so it outlives this
// process. wait returns when the installer exits. It only returns
// while this process is still alive, which means the update did not happen.
func (in Installer) Start(dir string, v Version) (wait func() error, err error) {
	cmd, err := in.command(dir, v)
	if err != nil {
		return nil, err
	}
	log, err := os.Create(filepath.Join(dir, logName))
	if err != nil {
		return nil, err
	}
	cmd.Stdout = log
	cmd.Stderr = log
	detach(cmd)
	if err := cmd.Start(); err != nil {
		log.Close()
		return nil, err
	}
	return func() error {
		err := cmd.Wait()
		log.Close()
		if err == nil {
			return errors.New("the installer finished but this Catsitter is still running")
		}
		if line := lastLine(filepath.Join(dir, logName)); line != "" {
			return errors.New(line)
		}
		return fmt.Errorf("the installer failed: %w", err)
	}, nil
}

// HandOff runs the installer for a terminal user. See handOff for each OS.
func (in Installer) HandOff(dir string, v Version) error {
	cmd, err := in.command(dir, v)
	if err != nil {
		return err
	}
	return handOff(cmd, dir)
}

// lastLine is the installer's last "catsitter install:" message, or else its
// last line. PowerShell follows a thrown message with error-record lines.
func lastLine(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	const tag = "catsitter install:"
	lines := strings.Split(strings.TrimSpace(string(bytes.ReplaceAll(b, []byte("\r"), nil))), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		if _, msg, ok := strings.Cut(lines[i], tag); ok {
			return strings.TrimSpace(msg)
		}
	}
	return strings.TrimSpace(lines[len(lines)-1])
}

// Installed reports whether exe is the copy the installer manages, so that
// installing over it updates the running app.
func Installed(exe string, h Home) bool {
	want := InstallPath(runtime.GOOS, h)
	if want == "" || exe == "" {
		return false
	}
	return samePath(resolve(exe), resolve(want), runtime.GOOS)
}

func resolve(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	if a, err := filepath.Abs(p); err == nil {
		p = a
	}
	return filepath.Clean(p)
}

// samePath compares paths, ignoring case where the file system usually does.
func samePath(a, b, goos string) bool {
	if goos == "windows" || goos == "darwin" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
