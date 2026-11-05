package browser

import (
	"slices"
	"testing"
)

func TestCommandPerOS(t *testing.T) {
	const url = "https://example.com/x"
	cases := map[string][]string{
		"darwin":  {"open", url},
		"windows": {"rundll32", "url.dll,FileProtocolHandler", url},
		"linux":   {"xdg-open", url},
	}
	for goos, want := range cases {
		if got := command(goos, url).Args; !slices.Equal(got, want) {
			t.Errorf("%s: %q, want %q", goos, got, want)
		}
	}
}

func TestOnlyGitHubPagesOpen(t *testing.T) {
	for _, raw := range []string{"https://github.com/xanderwasserman", "https://github.com/xanderwasserman/catsitter/releases/tag/v0.1.4"} {
		if !allowed(raw) {
			t.Errorf("%q refused", raw)
		}
	}
	for _, raw := range []string{
		"",
		"file:///C:/Windows/System32/calc.exe",
		`C:\Windows\System32\calc.exe`,
		"/usr/bin/xcalc",
		"http://github.com/xanderwasserman",
		"https://evil.example/",
		"https://github.com.evil.example/",
		"https://user@github.com/x",
		"https://github.com:8443/x",
		"https://github.com/x y",
	} {
		if allowed(raw) {
			t.Errorf("%q allowed", raw)
		}
	}
}
