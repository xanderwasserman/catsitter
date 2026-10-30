//go:build !windows

package app

import (
	"os"
	"syscall"
)

func listenSignals() []os.Signal {
	return []os.Signal{os.Interrupt, syscall.SIGTERM}
}
