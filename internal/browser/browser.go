// Package browser opens web pages in the user's default browser.
package browser

import (
	"log"
	"net/url"
	"os/exec"
	"runtime"
)

// Open shows url in the default browser. It returns at once. Only
// https://github.com pages are opened; anything else, such as a file path,
// is refused because the OS openers would run it.
func Open(raw string) {
	if !allowed(raw) {
		log.Printf("refusing to open %q", raw)
		return
	}
	cmd := command(runtime.GOOS, raw)
	if err := cmd.Start(); err != nil {
		log.Printf("open %s: %v", raw, err)
		return
	}
	go cmd.Wait()
}

func allowed(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == "github.com" && u.User == nil && u.Opaque == "" && u.String() == raw
}

func command(goos, url string) *exec.Cmd {
	switch goos {
	case "darwin":
		return exec.Command("open", url)
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return exec.Command("xdg-open", url)
	}
}
