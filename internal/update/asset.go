package update

import (
	"fmt"
	"path/filepath"
)

// ArchiveName is the release archive GoReleaser publishes for one platform.
func ArchiveName(v Version, goos, goarch string) (string, error) {
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", fmt.Errorf("no Catsitter release for %s/%s", goos, goarch)
	}
	switch goos {
	case "linux":
		return fmt.Sprintf("catsitter_%s_linux_%s.tar.gz", v, goarch), nil
	case "darwin", "windows":
		return fmt.Sprintf("catsitter_%s_%s_%s.zip", v, goos, goarch), nil
	default:
		return "", fmt.Errorf("no Catsitter release for %s/%s", goos, goarch)
	}
}

// Home holds the per-user directories the installers write to.
type Home struct {
	Dir          string
	LocalAppData string
}

// InstallPath is where the installer puts the executable on goos.
func InstallPath(goos string, h Home) string {
	switch goos {
	case "darwin":
		return filepath.Join(h.Dir, "Applications", "Catsitter.app", "Contents", "MacOS", "catsitter")
	case "windows":
		if h.LocalAppData == "" {
			return ""
		}
		return filepath.Join(h.LocalAppData, "Programs", "Catsitter", "catsitter.exe")
	default:
		return filepath.Join(h.Dir, ".local", "bin", "catsitter")
	}
}
