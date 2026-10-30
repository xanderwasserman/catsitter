package config

import (
	"testing"
	"time"
)

func TestParseDurationExamples(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"10m", 10 * time.Minute},
		{"2h", 2 * time.Hour},
		{"1.5h", 90 * time.Minute},
		{"2h30m", 2*time.Hour + 30*time.Minute},
		{"2h 30m", 2*time.Hour + 30*time.Minute},
		{"90s", 90 * time.Second},
		{"2 hours", 2 * time.Hour},
		{"2 hour", 2 * time.Hour},
		{"2 hrs", 2 * time.Hour},
		{"2 hr", 2 * time.Hour},
		{"1.5 hours", 90 * time.Minute},
		{"10 minutes", 10 * time.Minute},
		{"10 minute", 10 * time.Minute},
		{"10 mins", 10 * time.Minute},
		{"10 min", 10 * time.Minute},
		{"90 seconds", 90 * time.Second},
		{"90 second", 90 * time.Second},
		{"90 secs", 90 * time.Second},
		{"90 sec", 90 * time.Second},
		{"2h30m10s", 2*time.Hour + 30*time.Minute + 10*time.Second},
		{"", 0},
		{"0", 0},
		{"unlimited", 0},
		{"Forever", 0},
		{"until I stop it", 0},
	}
	for _, tc := range cases {
		got, err := ParseDuration(tc.in)
		if err != nil {
			t.Fatalf("ParseDuration(%q) error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("ParseDuration(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

func TestParseDurationRejects(t *testing.T) {
	for _, in := range []string{"10", "abc", "10x", "2h3", "1.2.3h", "-5m", "10m!"} {
		if _, err := ParseDuration(in); err == nil {
			t.Fatalf("ParseDuration(%q) succeeded", in)
		}
	}
}
