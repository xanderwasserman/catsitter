//go:build darwin

package mac

/*
#include <pthread.h>

static int csOnMainThread(void) {
	return pthread_main_np();
}
*/
import "C"
import (
	"fmt"
	"sync"
	"time"
)

type mainCall struct {
	fn   func() error
	done chan error
}

var (
	mainMu sync.Mutex
	mainQ  []mainCall
)

// RunOnMain runs fn on the main thread and waits for it.
// CoreGraphics input calls are made from that thread.
func RunOnMain(fn func() error) error {
	if fn == nil {
		return fmt.Errorf("missing main-thread function")
	}
	if C.csOnMainThread() != 0 {
		return fn()
	}
	call := mainCall{fn: fn, done: make(chan error, 1)}
	mainMu.Lock()
	mainQ = append(mainQ, call)
	mainMu.Unlock()
	select {
	case err := <-call.done:
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timed out waiting for the main thread")
	}
}

// Drain runs queued main-thread work. The Ebiten update loop calls it.
func Drain() {
	mainMu.Lock()
	batch := mainQ
	mainQ = nil
	mainMu.Unlock()
	for _, call := range batch {
		call.done <- call.fn()
	}
}
