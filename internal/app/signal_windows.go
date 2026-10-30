//go:build windows

package app

import "os"

func listenSignals() []os.Signal {
	return []os.Signal{os.Interrupt}
}
