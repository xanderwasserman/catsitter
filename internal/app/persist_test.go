package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/xanderwasserman/catsitter/internal/config"
	"github.com/xanderwasserman/catsitter/internal/host"
)

func TestOneOffFlagsStayOutOfTheFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	saved := config.Defaults()
	live := saved
	live.Interval = "5s"
	live.Pixels = 4
	live.HidePet = true
	a := New(live, saved, path, host.Backends{})
	a.Shutdown()
	got, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Interval != "10s" || got.Pixels != 1 || got.HidePet {
		t.Fatalf("saved %+v", got)
	}
}

func TestMenuChangeIsSavedWithoutTheOverlay(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	saved := config.Defaults()
	live := saved
	live.Interval = "5s"
	live.Pixels = 4
	a := New(live, saved, path, host.Backends{})
	a.SetMouse(false)
	a.SetInterval(30 * time.Second)
	if a.View().Interval != 30*time.Second || a.View().Mouse {
		t.Fatalf("live view %+v", a.View())
	}
	got, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.IntervalDuration() != 30*time.Second || got.Pixels != 1 || got.Mouse {
		t.Fatalf("saved %+v", got)
	}
}
