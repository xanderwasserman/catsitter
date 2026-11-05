package app

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/xanderwasserman/catsitter/internal/config"
	"github.com/xanderwasserman/catsitter/internal/host"
	"github.com/xanderwasserman/catsitter/internal/update"
)

type fakeUpdates struct {
	st               update.Status
	checks, installs int
	auto             []bool
}

func (f *fakeUpdates) Status() update.Status { return f.st }
func (f *fakeUpdates) Run(context.Context)   {}
func (f *fakeUpdates) CheckNow()             { f.checks++ }
func (f *fakeUpdates) Install()              { f.installs++ }
func (f *fakeUpdates) SetAuto(v bool)        { f.auto = append(f.auto, v) }

func TestUpdatesReachTheViewAndTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	up := &fakeUpdates{st: update.Status{Version: "0.1.2", State: update.Available}}
	a := New(config.Defaults(), config.Defaults(), path, host.Backends{}, up)
	if v := a.View(); v.Update.State != update.Available || v.Update.Version != "0.1.2" {
		t.Fatalf("view %+v", v.Update)
	}
	a.CheckForUpdates()
	a.InstallUpdate()
	a.SetAutoUpdate(false)
	if up.checks != 1 || up.installs != 1 || len(up.auto) != 1 || up.auto[0] {
		t.Fatalf("calls %+v", up)
	}
	got, err := config.Load(path)
	if err != nil || got.CheckUpdates {
		t.Fatalf("saved %+v %v", got, err)
	}
	before := a.View().AboutAsks
	a.ShowAbout()
	if a.View().AboutAsks != before+1 {
		t.Error("ShowAbout did not reach the view")
	}
}
