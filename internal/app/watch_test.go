package app

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/xanderwasserman/catsitter/internal/config"
	"github.com/xanderwasserman/catsitter/internal/host"
	"github.com/xanderwasserman/catsitter/internal/scheduler"
)

type stepClock struct{ t time.Time }

func (c *stepClock) Now() time.Time { return c.t }

type heldInhibit struct{ on bool }

func (h *heldInhibit) Acquire() error { h.on = true; return nil }
func (h *heldInhibit) Release() error { h.on = false; return nil }

// watchApp is a session with a one-second watch, driven by a fake clock.
func watchApp(t *testing.T) (*App, *stepClock, *heldInhibit, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	cfg := config.Defaults()
	cfg.Mouse, cfg.Shift, cfg.Switch = false, false, false
	cfg.Duration = "1s"
	a := New(cfg, cfg, path, host.Backends{}, nil)
	clock := &stepClock{t: time.Unix(0, 0)}
	inh := &heldInhibit{}
	a.sched = scheduler.New(nil, nil, inh, clock, nil, nil)
	a.push()
	return a, clock, inh, path
}

func (c *stepClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func TestWatchLimitPausesInsteadOfQuitting(t *testing.T) {
	a, clock, inh, path := watchApp(t)
	a.step()
	if !inh.on || a.View().Paused {
		t.Fatalf("watch did not start: held=%v paused=%v", inh.on, a.View().Paused)
	}
	clock.advance(1200 * time.Millisecond)
	a.step()
	v := a.View()
	if !v.Paused || v.Quitting {
		t.Fatalf("after the limit: paused=%v quitting=%v", v.Paused, v.Quitting)
	}
	if inh.on {
		t.Fatal("the inhibitor is still held after the watch ended")
	}
	if v.Duration != time.Second {
		t.Fatalf("watch length changed to %v", v.Duration)
	}
	clock.advance(time.Hour)
	a.step()
	if v := a.View(); !v.Paused || v.Quitting || inh.on {
		t.Fatalf("an hour later: paused=%v quitting=%v held=%v", v.Paused, v.Quitting, inh.on)
	}
	a.Shutdown()
	got, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.LimitDuration() != time.Second {
		t.Fatalf("saved watch %q", got.Duration)
	}
}

func TestResumeAfterTheWatchStartsAFreshOne(t *testing.T) {
	a, clock, inh, _ := watchApp(t)
	a.step()
	clock.advance(1200 * time.Millisecond)
	a.step()
	if !a.View().Paused {
		t.Fatal("the watch did not end")
	}

	a.TogglePause()
	clock.advance(200 * time.Millisecond)
	a.step()
	if a.View().Paused || !inh.on {
		t.Fatalf("resume ended at once: paused=%v held=%v", a.View().Paused, inh.on)
	}
	clock.advance(600 * time.Millisecond)
	a.step()
	if a.View().Paused {
		t.Fatal("the fresh watch ended early")
	}
	if a.View().Duration != time.Second {
		t.Fatalf("watch length changed to %v", a.View().Duration)
	}
	clock.advance(600 * time.Millisecond)
	a.step()
	if !a.View().Paused || inh.on {
		t.Fatalf("the fresh watch did not end: paused=%v held=%v", a.View().Paused, inh.on)
	}
}

func TestUnlimitedWatchNeverPauses(t *testing.T) {
	a, clock, inh, _ := watchApp(t)
	a.SetDuration(0)
	a.step()
	clock.advance(24 * time.Hour)
	a.step()
	if a.View().Paused || !inh.on {
		t.Fatalf("unlimited watch: paused=%v held=%v", a.View().Paused, inh.on)
	}
}
