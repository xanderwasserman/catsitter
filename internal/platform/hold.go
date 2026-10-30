package platform

import "fmt"

// HoldAndTap presses a modifier, taps another key count times, and releases
// the modifier. The modifier release runs even when a tap fails.
// afterHold, afterDown, and afterUp run between events. A nil pause is skipped.
func HoldAndTap(holdDown, holdUp, keyDown, keyUp func() error, count int, afterHold, afterDown, afterUp func()) (err error) {
	if holdDown == nil || holdUp == nil || keyDown == nil || keyUp == nil {
		return fmt.Errorf("missing key sender")
	}
	if count < 1 {
		count = 1
	}
	pause := func(fn func()) {
		if fn != nil {
			fn()
		}
	}
	if err = holdDown(); err != nil {
		return err
	}
	defer func() {
		if relErr := holdUp(); relErr != nil && err == nil {
			err = relErr
		}
	}()
	pause(afterHold)
	for i := 0; i < count; i++ {
		if err = keyDown(); err != nil {
			return err
		}
		pause(afterDown)
		if err = keyUp(); err != nil {
			return err
		}
		pause(afterUp)
	}
	return nil
}
