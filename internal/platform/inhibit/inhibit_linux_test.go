//go:build linux

package inhibit

import (
	"errors"
	"testing"
)

func TestDropReleasedKeepsFailures(t *testing.T) {
	tries := 0
	left, err := dropReleased([]func() error{
		func() error { return nil },
		func() error {
			tries++
			if tries == 1 {
				return errors.New("busy")
			}
			return nil
		},
	})
	if err == nil || err.Error() != "busy" || len(left) != 1 {
		t.Fatalf("err=%v left=%d", err, len(left))
	}
	left, err = dropReleased(left)
	if err != nil || len(left) != 0 {
		t.Fatalf("retry left=%d err=%v", len(left), err)
	}
}
