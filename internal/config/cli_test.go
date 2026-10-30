package config

import (
	"bytes"
	"testing"
	"time"
)

func TestApplyFlags(t *testing.T) {
	f, err := ParseFlags([]string{"catsitter", "--seconds", "15", "--pixels", "3", "--mode", "mks", "--tabs", "0", "--key", "cmd", "--duration", "1.5h", "--no-pet", "--no-inhibit"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := f.Apply(Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IntervalDuration() != 15*time.Second || cfg.Pixels != 3 {
		t.Fatalf("%+v", cfg)
	}
	if !cfg.Mouse || !cfg.Shift || !cfg.Switch {
		t.Fatalf("mode mks: %+v", cfg)
	}
	if cfg.Tabs != 0 || cfg.Modifier != "cmd" || !cfg.HidePet || cfg.Inhibit {
		t.Fatalf("%+v", cfg)
	}
	if cfg.LimitDuration() != 90*time.Minute {
		t.Fatalf("limit %s", cfg.LimitDuration())
	}
}

func TestModeOverriddenByActionFlag(t *testing.T) {
	f, err := ParseFlags([]string{"catsitter", "--mode", "m", "--shift"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := f.Apply(Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Mouse || !cfg.Shift || cfg.Switch {
		t.Fatalf("%+v", cfg)
	}
}

func TestUnsetFlagsKeepConfig(t *testing.T) {
	base := Defaults()
	base.Interval = "45s"
	base.Switch = true
	base.Tabs = 2
	f, err := ParseFlags([]string{"catsitter"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := f.Apply(base)
	if err != nil {
		t.Fatal(err)
	}
	if cfg != base {
		t.Fatalf("got %+v", cfg)
	}
}

func TestRejectsBadMode(t *testing.T) {
	f, err := ParseFlags([]string{"catsitter", "--mode", "mx"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Apply(Defaults()); err == nil {
		t.Fatal("expected error")
	}
}

func TestVersionFlag(t *testing.T) {
	f, err := ParseFlags([]string{"catsitter", "--version"}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if !f.Version {
		t.Fatal("version not set")
	}
}
