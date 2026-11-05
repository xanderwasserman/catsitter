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
