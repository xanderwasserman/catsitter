package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// Config is the on-disk settings file.
type Config struct {
	Interval  string `json:"interval"`
	Pixels    int    `json:"pixels"`
	Tabs      int    `json:"tabs"`
	Inhibit   bool   `json:"inhibit"`
	Mouse     bool   `json:"mouse"`
	Shift     bool   `json:"shift"`
	Switch    bool   `json:"switch"`
	Modifier  string `json:"modifier,omitempty"`
	Duration  string `json:"duration,omitempty"`
	HidePet   bool   `json:"hidePet"`
	WindowX   int    `json:"windowX"`
	WindowY   int    `json:"windowY"`
	WindowSet bool   `json:"windowSet"`
}

// Defaults are the first-run settings.
func Defaults() Config {
	return Config{
		Interval: "10s",
		Pixels:   1,
		Tabs:     1,
		Inhibit:  true,
		Mouse:    true,
		Shift:    true,
		Switch:   false,
	}
}

// Path is the config file in the OS config directory.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "catsitter", "config.json"), nil
}

// Load reads the config file. A missing file returns Defaults.
func Load(path string) (Config, error) {
	cfg := Defaults()
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Save writes the config file, creating the directory if needed.
func Save(path string, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o600)
}

// Validate checks values the scheduler will rely on.
func (c Config) Validate() error {
	d, err := ParseDuration(c.Interval)
	if err != nil || d < time.Second {
		return fmt.Errorf("interval must be at least 1s, got %q", c.Interval)
	}
	if _, err := ParseDuration(c.Duration); err != nil {
		return fmt.Errorf("duration: %w", err)
	}
	if c.Pixels < 1 {
		return fmt.Errorf("pixels must be at least 1")
	}
	if c.Tabs < 0 {
		return fmt.Errorf("tabs must be zero or more")
	}
	switch normalizeModifier(c.Modifier) {
	case "", "alt", "cmd":
	default:
		return fmt.Errorf("modifier must be alt or cmd, got %q", c.Modifier)
	}
	return nil
}

// IntervalDuration parses Interval.
func (c Config) IntervalDuration() time.Duration {
	d, err := ParseDuration(c.Interval)
	if err != nil || d < time.Second {
		return 10 * time.Second
	}
	return d
}

// LimitDuration parses Duration. Zero means unlimited.
func (c Config) LimitDuration() time.Duration {
	d, _ := ParseDuration(c.Duration)
	return d
}

// ModifierKey is the window-switch modifier, defaulting per OS.
func (c Config) ModifierKey() string {
	if m := normalizeModifier(c.Modifier); m != "" {
		return m
	}
	if runtime.GOOS == "darwin" {
		return "cmd"
	}
	return "alt"
}

func normalizeModifier(m string) string {
	switch m {
	case "", "alt", "option":
		if m == "" {
			return ""
		}
		return "alt"
	case "cmd", "command", "super", "meta", "win":
		return "cmd"
	default:
		return m
	}
}
