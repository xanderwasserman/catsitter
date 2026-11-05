// Package browser opens web pages in the user's default browser.
package browser

import (
	"log"
	"os/exec"
	"runtime"
)

// Open shows url in the default browser. It returns at once.
func Open(url string) {
	cmd := command(runtime.GOOS, url)
	if err := cmd.Start(); err != nil {
		log.Printf("open %s: %v", url, err)
		return
	}
	go cmd.Wait()
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
