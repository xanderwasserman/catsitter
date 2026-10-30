package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "catsitter", "config.json")
	cfg := Defaults()
	cfg.Interval = "30s"
	cfg.Pixels = 4
	cfg.Tabs = 0
	cfg.Switch = true
	cfg.Modifier = "cmd"
	cfg.Duration = "2h30m"
	cfg.HidePet = true
	cfg.WindowX = 40
	cfg.WindowY = 80
	cfg.WindowSet = true
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != cfg {
		t.Fatalf("got %+v want %+v", got, cfg)
	}
	if got.IntervalDuration() != 30*time.Second {
		t.Fatalf("interval %s", got.IntervalDuration())
	}
	if got.LimitDuration() != 2*time.Hour+30*time.Minute {
		t.Fatalf("limit %s", got.LimitDuration())
	}
	if got.ModifierKey() != "cmd" {
		t.Fatalf("modifier %s", got.ModifierKey())
	}
}

func TestLoadMissingReturnsDefaults(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "nope.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got != Defaults() {
		t.Fatalf("%+v", got)
	}
	if got.IntervalDuration() != 10*time.Second {
		t.Fatal(got.IntervalDuration())
	}
	if got.LimitDuration() != 0 {
		t.Fatal(got.LimitDuration())
	}
}

func TestValidate(t *testing.T) {
	cfg := Defaults()
	cfg.Interval = "nope"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected interval error")
	}
	cfg = Defaults()
	cfg.Pixels = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected pixels error")
	}
	cfg = Defaults()
	cfg.Duration = "10"
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected duration error")
	}
}

func TestLoadCorrupt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected error")
	}
}
