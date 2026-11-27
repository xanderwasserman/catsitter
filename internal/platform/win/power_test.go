package win

import (
	"errors"
	"testing"
)

type fakePower struct {
	creates   int
	active    map[PowerKind]int
	failSet   map[PowerKind]bool
	failClear map[PowerKind]bool
	failNew   bool
}

func newFakePower() *fakePower {
	return &fakePower{active: map[PowerKind]int{}, failSet: map[PowerKind]bool{}, failClear: map[PowerKind]bool{}}
}

func (f *fakePower) Create() (uintptr, error) {
	if f.failNew {
		return 0, errors.New("no handle")
	}
	f.creates++
	return 7, nil
}

func (f *fakePower) Set(h uintptr, k PowerKind) error {
	if h != 7 {
		return errors.New("bad handle")
	}
	if f.failSet[k] {
		return errors.New("set refused")
	}
	f.active[k]++
	return nil
}

func (f *fakePower) Clear(h uintptr, k PowerKind) error {
	if h != 7 {
		return errors.New("bad handle")
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

func (f *fakePower) want(t *testing.T, display, system int) {
	t.Helper()
	if f.active[PowerDisplayRequired] != display || f.active[PowerSystemRequired] != system {
		t.Fatalf("display %d system %d, want %d %d", f.active[PowerDisplayRequired], f.active[PowerSystemRequired], display, system)
	}
}

func TestPowerAcquireAndRelease(t *testing.T) {
	api := newFakePower()
	p := NewPowerInhibitor(api)
	if err := p.Acquire(); err != nil {
		t.Fatal(err)
	}
	api.want(t, 1, 1)
	if err := p.Release(); err != nil {
		t.Fatal(err)
	}
	api.want(t, 0, 0)
}

func TestPowerRepeatedCallsNeitherStackNorLeak(t *testing.T) {
	api := newFakePower()
	p := NewPowerInhibitor(api)
	for range 5 {
		if err := p.Acquire(); err != nil {
			t.Fatal(err)
		}
		if err := p.Acquire(); err != nil {
			t.Fatal(err)
		}
		api.want(t, 1, 1)
		if err := p.Release(); err != nil {
			t.Fatal(err)
		}
		if err := p.Release(); err != nil {
			t.Fatal(err)
		}
		api.want(t, 0, 0)
	}
	if api.creates != 1 {
		t.Fatalf("created %d handles", api.creates)
	}
}

func TestPowerReleaseFailureIsReportedAndRetried(t *testing.T) {
	api := newFakePower()
	p := NewPowerInhibitor(api)
	if err := p.Acquire(); err != nil {
		t.Fatal(err)
	}
	api.failClear[PowerDisplayRequired] = true
	if err := p.Release(); err == nil {
		t.Fatal("release reported success with the display request still set")
	}
	api.want(t, 1, 0)
	api.failClear[PowerDisplayRequired] = false
	if err := p.Release(); err != nil {
		t.Fatal(err)
	}
	api.want(t, 0, 0)
}

func TestPowerAcquireFailureRollsBack(t *testing.T) {
	api := newFakePower()
	api.failSet[PowerSystemRequired] = true
	p := NewPowerInhibitor(api)
	if err := p.Acquire(); err == nil {
		t.Fatal("acquire reported success")
	}
	api.want(t, 0, 0)
	api.failSet[PowerSystemRequired] = false
	if err := p.Acquire(); err != nil {
		t.Fatal(err)
	}
	api.want(t, 1, 1)
}

func TestPowerCreateFailure(t *testing.T) {
	api := newFakePower()
	api.failNew = true
	p := NewPowerInhibitor(api)
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
	api.want(t, 1, 1)
}
