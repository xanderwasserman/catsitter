//go:build windows

package app

import (
	"os"
	"syscall"
)

func listenSignals() []os.Signal {
	// Interrupt is Ctrl+C and Ctrl+Break.
	// SIGTERM is console close, logoff, and shutdown.
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
