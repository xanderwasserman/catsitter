package control

import (
	"testing"

	"github.com/xanderwasserman/catsitter/internal/update"
)

func TestUpdateItem(t *testing.T) {
	v, _ := update.ParseVersion("0.1.3")
	base := update.Status{Version: "0.1.2", Enabled: true, CanInstall: true, Latest: update.Release{Version: v}}
	cases := []struct {
		name   string
		edit   func(*update.Status)
		label  string
		action UpdateAction
	}{
		{"dev", func(s *update.Status) { s.Enabled = false }, "", UpdateNone},
		{"idle", func(s *update.Status) {}, "Check for updates", UpdateCheck},
		{"checking", func(s *update.Status) { s.Checking = true }, "Checking for updates...", UpdateNone},
		{"available", func(s *update.Status) { s.State = update.Available }, "Update to 0.1.3", UpdateInstall},
		{"retry", func(s *update.Status) { s.State = update.Failed }, "Update to 0.1.3", UpdateInstall},
		{"foreign", func(s *update.Status) { s.State = update.Available; s.CanInstall = false }, "Get Catsitter 0.1.3...", UpdateOpenPage},
		{"download", func(s *update.Status) { s.State = update.Downloading; s.Progress = 0.42 }, "Updating... 42%", UpdateNone},
		{"install", func(s *update.Status) { s.State = update.Installing }, "Installing 0.1.3...", UpdateNone},
	}
	for _, c := range cases {
		st := base
		c.edit(&st)
		label, action := UpdateItem(st)
		if label != c.label || action != c.action {
			t.Errorf("%s: %q %v, want %q %v", c.name, label, action, c.label, c.action)
		}
	}
}

func TestVersionLine(t *testing.T) {
	if got := VersionLine("0.1.2"); got != "Catsitter 0.1.2 by @xanderwasserman" {
		t.Error(got)
	}
	if got := VersionLine("dev"); got != "Catsitter dev build by @xanderwasserman" {
		t.Error(got)
	}
}
