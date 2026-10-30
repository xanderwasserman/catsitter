package config

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

// Flags are the command-line overrides. Zero values mean "not set"
// except for the boolean pairs, which use Visited.
type Flags struct {
	Seconds   int
	Pixels    int
	Mode      string
	Tabs      int
	Key       string
	Duration  string
	NoPet     bool
	Version   bool
	Inhibit   bool
	NoInhibit bool
	Mouse     bool
	NoMouse   bool
	Shift     bool
	NoShift   bool
	Switch    bool
	NoSwitch  bool

	setSeconds   bool
	setPixels    bool
	setMode      bool
	setTabs      bool
	setKey       bool
	setDuration  bool
	setNoPet     bool
	setInhibit   bool
	setNoInhibit bool
	setMouse     bool
	setNoMouse   bool
	setShift     bool
	setNoShift   bool
	setSwitch    bool
	setNoSwitch  bool
}

// ParseFlags parses argv. The first argument is the program name.
func ParseFlags(argv []string, out io.Writer) (Flags, error) {
	fs := flag.NewFlagSet("catsitter", flag.ContinueOnError)
	fs.SetOutput(out)
	var f Flags
	fs.IntVar(&f.Seconds, "seconds", 0, "idle seconds before a nudge (default 10)")
	fs.IntVar(&f.Pixels, "pixels", 0, "mouse jiggle distance in pixels (default 1)")
	fs.StringVar(&f.Mode, "mode", "", "actions to enable: m mouse, k shift, s switch windows (default mk)")
	fs.IntVar(&f.Tabs, "tabs", 0, "window switches per nudge; 0 means a random 1 to 3 (default 1)")
	fs.StringVar(&f.Key, "key", "", "switch modifier: alt or cmd (default cmd on macOS, alt elsewhere)")
	fs.StringVar(&f.Duration, "duration", "", "how long to keep watch, for example 10m, 2h, 1.5h, 2h30m, 90s")
	fs.BoolVar(&f.NoPet, "no-pet", false, "hide the cat and keep working")
	fs.BoolVar(&f.Version, "version", false, "print the version and exit")
	fs.BoolVar(&f.Inhibit, "inhibit", false, "keep the system and display awake")
	fs.BoolVar(&f.NoInhibit, "no-inhibit", false, "do not hold a sleep inhibitor")
	fs.BoolVar(&f.Mouse, "mouse", false, "enable the mouse jiggle")
	fs.BoolVar(&f.NoMouse, "no-mouse", false, "disable the mouse jiggle")
	fs.BoolVar(&f.Shift, "shift", false, "enable the Shift key tap")
	fs.BoolVar(&f.NoShift, "no-shift", false, "disable the Shift key tap")
	fs.BoolVar(&f.Switch, "switch", false, "enable Alt or Cmd+Tab window switching")
	fs.BoolVar(&f.NoSwitch, "no-switch", false, "disable window switching")
	if err := fs.Parse(argv[1:]); err != nil {
		return Flags{}, err
	}
	if fs.NArg() != 0 {
		return Flags{}, fmt.Errorf("unexpected argument %q", fs.Arg(0))
	}
	fs.Visit(func(fl *flag.Flag) {
		switch fl.Name {
		case "seconds":
			f.setSeconds = true
		case "pixels":
			f.setPixels = true
		case "mode":
			f.setMode = true
		case "tabs":
			f.setTabs = true
		case "key":
			f.setKey = true
		case "duration":
			f.setDuration = true
		case "no-pet":
			f.setNoPet = true
		case "inhibit":
			f.setInhibit = true
		case "no-inhibit":
			f.setNoInhibit = true
		case "mouse":
			f.setMouse = true
		case "no-mouse":
			f.setNoMouse = true
		case "shift":
			f.setShift = true
		case "no-shift":
			f.setNoShift = true
		case "switch":
			f.setSwitch = true
		case "no-switch":
			f.setNoSwitch = true
		}
	})
	return f, nil
}

// Apply overlays flags that were actually passed onto cfg.
func (f Flags) Apply(cfg Config) (Config, error) {
	if f.setSeconds {
		if f.Seconds < 1 {
			return Config{}, fmt.Errorf("--seconds must be at least 1")
		}
		cfg.Interval = fmt.Sprintf("%ds", f.Seconds)
	}
	if f.setPixels {
		if f.Pixels < 1 {
			return Config{}, fmt.Errorf("--pixels must be at least 1")
		}
		cfg.Pixels = f.Pixels
	}
	if f.setMode {
		mode := strings.ToLower(strings.TrimSpace(f.Mode))
		cfg.Mouse, cfg.Shift, cfg.Switch = false, false, false
		for _, r := range mode {
			switch r {
			case 'm':
				cfg.Mouse = true
			case 'k':
				cfg.Shift = true
			case 's':
				cfg.Switch = true
			default:
				return Config{}, fmt.Errorf("--mode accepts only m, k, and s")
			}
		}
	}
	if f.setTabs {
		if f.Tabs < 0 {
			return Config{}, fmt.Errorf("--tabs must be zero or more")
		}
		cfg.Tabs = f.Tabs
	}
	if f.setKey {
		switch normalizeModifier(f.Key) {
		case "alt":
			cfg.Modifier = "alt"
		case "cmd":
			cfg.Modifier = "cmd"
		default:
			return Config{}, fmt.Errorf("--key must be alt or cmd")
		}
	}
	if f.setDuration {
		if _, err := ParseDuration(f.Duration); err != nil {
			return Config{}, fmt.Errorf("--duration: %w", err)
		}
		if isUnlimited(strings.TrimSpace(f.Duration)) {
			cfg.Duration = ""
		} else {
			d, _ := ParseDuration(f.Duration)
			cfg.Duration = FormatDuration(d)
			if cfg.Duration == "" {
				cfg.Duration = f.Duration
			}
		}
	}
	if f.setNoPet && f.NoPet {
		cfg.HidePet = true
	}
	if f.setNoInhibit && f.setInhibit {
		return Config{}, fmt.Errorf("pass only one of --inhibit and --no-inhibit")
	}
	if f.setNoMouse && f.setMouse {
		return Config{}, fmt.Errorf("pass only one of --mouse and --no-mouse")
	}
	if f.setNoShift && f.setShift {
		return Config{}, fmt.Errorf("pass only one of --shift and --no-shift")
	}
	if f.setNoSwitch && f.setSwitch {
		return Config{}, fmt.Errorf("pass only one of --switch and --no-switch")
	}
	if f.setNoInhibit {
		cfg.Inhibit = false
	}
	if f.setInhibit {
		cfg.Inhibit = true
	}
	if f.setNoMouse {
		cfg.Mouse = false
	}
	if f.setMouse {
		cfg.Mouse = true
	}
	if f.setNoShift {
		cfg.Shift = false
	}
	if f.setShift {
		cfg.Shift = true
	}
	if f.setNoSwitch {
		cfg.Switch = false
	}
	if f.setSwitch {
		cfg.Switch = true
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// SecondsDuration is a convenience for tests and the menu.
func Seconds(n int) time.Duration { return time.Duration(n) * time.Second }
