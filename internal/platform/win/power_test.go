package win

import (
	"errors"
	"testing"
)

// fakePower models one handle at a time. Closing it drops its requests.
type fakePower struct {
	creates   int
	open      uintptr
	active    map[powerKind]int
	failSet   map[powerKind]bool
	failClear map[powerKind]bool
	failNew   bool
	failClose bool
}

func newFakePower() *fakePower {
	return &fakePower{active: map[powerKind]int{}, failSet: map[powerKind]bool{}, failClear: map[powerKind]bool{}}
}

func (f *fakePower) create() (uintptr, error) {
	if f.failNew {
		return 0, errors.New("no handle")
	}
	if f.open != 0 {
		return 0, errors.New("handle leaked")
	}
	f.creates++
	f.open = uintptr(f.creates)
	return f.open, nil
}

func (f *fakePower) check(h uintptr) error {
	if h == 0 || h != f.open {
		return errors.New("bad handle")
	}
	return nil
}

func (f *fakePower) set(h uintptr, k powerKind) error {
	if err := f.check(h); err != nil {
		return err
	}
	if f.failSet[k] {
		return errors.New("set refused")
	}
	f.active[k]++
	return nil
}

func (f *fakePower) clear(h uintptr, k powerKind) error {
	if err := f.check(h); err != nil {
		return err
	}
	if f.failClear[k] {
		return errors.New("clear refused")
	}
	if f.active[k] == 0 {
		return errors.New("not set")
	}
	f.active[k]--
	return nil
}

func (f *fakePower) close(h uintptr) error {
	if err := f.check(h); err != nil {
		return err
	}
	if f.failClose {
		return errors.New("close refused")
	}
	f.open = 0
	clear(f.active)
	return nil
}

func (f *fakePower) want(t *testing.T, display, system int, open bool) {
	t.Helper()
	if f.active[powerDisplayRequired] != display || f.active[powerSystemRequired] != system || (f.open != 0) != open {
		t.Fatalf("display %d system %d open %v, want %d %d %v",
			f.active[powerDisplayRequired], f.active[powerSystemRequired], f.open != 0, display, system, open)
	}
}

func TestPowerAcquireAndRelease(t *testing.T) {
	api := newFakePower()
	p := newPowerInhibitor(api)
	if err := p.Acquire(); err != nil {
		t.Fatal(err)
	}
	api.want(t, 1, 1, true)
	if err := p.Release(); err != nil {
		t.Fatal(err)
	}
	api.want(t, 0, 0, false)
}

func TestPowerRepeatedCallsNeitherStackNorLeak(t *testing.T) {
	api := newFakePower()
	p := newPowerInhibitor(api)
	for range 5 {
		if err := p.Acquire(); err != nil {
			t.Fatal(err)
		}
		if err := p.Acquire(); err != nil {
			t.Fatal(err)
		}
		api.want(t, 1, 1, true)
		if err := p.Release(); err != nil {
			t.Fatal(err)
		}
		if err := p.Release(); err != nil {
			t.Fatal(err)
		}
		api.want(t, 0, 0, false)
	}
}

func TestPowerClearFailureIsReportedAndClosingDropsTheRequest(t *testing.T) {
	api := newFakePower()
	p := newPowerInhibitor(api)
	if err := p.Acquire(); err != nil {
		t.Fatal(err)
	}
	api.failClear[powerDisplayRequired] = true
	if err := p.Release(); err == nil {
		t.Fatal("release reported success after a clear failed")
	}
	api.want(t, 0, 0, false)
	if err := p.Release(); err != nil {
		t.Fatalf("second release: %v", err)
	}
}

func TestPowerCloseFailureKeepsStateForRetry(t *testing.T) {
	api := newFakePower()
	p := newPowerInhibitor(api)
	if err := p.Acquire(); err != nil {
		t.Fatal(err)
	}
	api.failClear[powerDisplayRequired] = true
	api.failClose = true
	if err := p.Release(); err == nil {
		t.Fatal("release reported success with the handle open")
	}
	api.want(t, 1, 0, true)
	api.failClear[powerDisplayRequired] = false
	api.failClose = false
	if err := p.Release(); err != nil {
		t.Fatal(err)
	}
	api.want(t, 0, 0, false)
}

func TestPowerAcquireFailureRollsBack(t *testing.T) {
	api := newFakePower()
	api.failSet[powerSystemRequired] = true
	p := newPowerInhibitor(api)
	if err := p.Acquire(); err == nil {
		t.Fatal("acquire reported success")
	}
	api.want(t, 0, 0, false)
	api.failSet[powerSystemRequired] = false
	if err := p.Acquire(); err != nil {
		t.Fatal(err)
	}
	api.want(t, 1, 1, true)
}

// A failed set whose rollback clear also fails must not leave the display
// request behind: closing the handle drops it.
func TestPowerAcquireFailureWithFailedRollbackHoldsNothing(t *testing.T) {
	api := newFakePower()
	api.failSet[powerSystemRequired] = true
	api.failClear[powerDisplayRequired] = true
	p := newPowerInhibitor(api)
	if err := p.Acquire(); err == nil {
		t.Fatal("acquire reported success")
	}
	api.want(t, 0, 0, false)
	if err := p.Release(); err != nil {
		t.Fatalf("release after rollback: %v", err)
	}
}

func TestPowerCreateFailure(t *testing.T) {
	api := newFakePower()
	api.failNew = true
	p := newPowerInhibitor(api)
	if err := p.Acquire(); err == nil {
		t.Fatal("acquire reported success")
	}
	if err := p.Release(); err != nil {
		t.Fatalf("release with nothing held: %v", err)
	}
	api.failNew = false
	if err := p.Acquire(); err != nil {
		t.Fatal(err)
	}
	api.want(t, 1, 1, true)
}
