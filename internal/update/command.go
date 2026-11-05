package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
)

// Terminal runs --check-update and --update.
type Terminal struct {
	Options
	// Installer replaces this copy. HandOff does not return on success on macOS and Linux.
	Installer   Installer
	InstallPath string
	Out         io.Writer
}

// Check prints whether a newer release exists.
func (t Terminal) Check(ctx context.Context) (Release, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, checkTimeout)
	defer cancel()
	rel, _, err := t.Releases.Latest(ctx, "")
	if err != nil {
		return Release{}, false, fmt.Errorf("could not check for updates: %w", err)
	}
	switch {
	case !IsRelease(t.Version):
		fmt.Fprintf(t.Out, "This is a dev build (%s). The latest release is %s.\n", t.Version, rel.Version)
		return rel, false, nil
	case Offer(t.Version, rel.Version):
		fmt.Fprintf(t.Out, "Catsitter %s is out. You have %s.\n", rel.Version, t.Version)
		return rel, true, nil
	default:
		fmt.Fprintf(t.Out, "Catsitter %s is up to date.\n", t.Version)
		return rel, false, nil
	}
}

// Install checks, then downloads, verifies and installs the newer release.
func (t Terminal) Install(ctx context.Context) error {
	rel, newer, err := t.Check(ctx)
	if err != nil || !newer {
		return err
	}
	if !IsRelease(t.Version) {
		return errors.New("dev builds do not update themselves")
	}
	if !t.CanInstall {
		return fmt.Errorf("this copy is not the installed one at %s, so re-run the install script to update", t.InstallPath)
	}
	ctx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()
	last := -1
	dir, err := t.Releases.Stage(ctx, rel, t.GOOS, t.GOARCH, func(done, total int64) {
		if total <= 0 {
			return
		}
		if pct := int(done * 100 / total); pct != last {
			last = pct
			fmt.Fprintf(t.Out, "\rDownloading Catsitter %s... %d%%", rel.Version, pct)
		}
	})
	if last >= 0 {
		fmt.Fprintln(t.Out)
	}
	if err != nil {
		return err
	}
	fmt.Fprintln(t.Out, "The download matches checksums.txt.")
	rec := t.Store.Load()
	rec.Pending = &Pending{From: t.Version, To: rel.Version.String(), Dir: dir}
	if t.Store.Path != "" {
		_ = t.Store.Save(rec)
	}
	if err := t.Installer.HandOff(dir, rel.Version); err != nil {
		os.RemoveAll(dir)
		return err
	}
	return nil
}
