package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// token matches one number plus a unit. Longer names come first so "hours"
// is not read as "h" plus leftover text.
var durationToken = regexp.MustCompile(`(?i)^(\d+(?:\.\d+)?)[\t ]*(hours|hour|hrs|hr|h|minutes|minute|mins|min|m|seconds|second|secs|sec|s)`)

// ParseDuration parses the keep-awake limit.
// Empty, "0", "unlimited", "forever", and "until I stop it" mean no limit.
// Combined values such as 2h30m and fractional values such as 1.5h are accepted.
func ParseDuration(raw string) (time.Duration, error) {
	s := strings.TrimSpace(raw)
	if isUnlimited(s) {
		return 0, nil
	}
	var total time.Duration
	matched := false
	rest := s
	for strings.TrimSpace(rest) != "" {
		rest = strings.TrimSpace(rest)
		m := durationToken.FindStringSubmatch(rest)
		if m == nil {
			return 0, fmt.Errorf("invalid duration %q", raw)
		}
		n, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", raw)
		}
		if n < 0 {
			return 0, fmt.Errorf("invalid duration %q", raw)
		}
		unit, err := unitDuration(m[2])
		if err != nil {
			return 0, err
		}
		total += time.Duration(n * float64(unit))
		rest = rest[len(m[0]):]
		matched = true
	}
	if !matched {
		return 0, fmt.Errorf("invalid duration %q", raw)
	}
	return total, nil
}

func isUnlimited(s string) bool {
	switch strings.ToLower(s) {
	case "", "0", "unlimited", "forever", "until i stop it", "until i stop":
		return true
	default:
		return false
	}
}

func unitDuration(unit string) (time.Duration, error) {
	switch strings.ToLower(unit) {
	case "h", "hr", "hrs", "hour", "hours":
		return time.Hour, nil
	case "m", "min", "mins", "minute", "minutes":
		return time.Minute, nil
	case "s", "sec", "secs", "second", "seconds":
		return time.Second, nil
	default:
		return 0, fmt.Errorf("unknown duration unit %q", unit)
	}
}

// FormatDuration renders a duration the way the menu and config file show it.
// Zero means unlimited.
func FormatDuration(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	if d%time.Hour == 0 {
		return fmt.Sprintf("%dh", int(d/time.Hour))
	}
	if d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	if d%time.Second == 0 {
		h := d / time.Hour
		m := (d % time.Hour) / time.Minute
		if h > 0 {
			return fmt.Sprintf("%dh%dm", int(h), int(m))
		}
		s := (d % time.Minute) / time.Second
		if m > 0 && s == 0 {
			return fmt.Sprintf("%dm", int(m))
		}
		return fmt.Sprintf("%ds", int(d/time.Second))
	}
	return d.String()
}
