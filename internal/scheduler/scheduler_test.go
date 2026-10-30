package scheduler

import (
	"errors"
	"testing"
	"time"
)

type fakeClock struct{ t time.Time }

func (f *fakeClock) Now() time.Time { return f.t }

type fakeIdle struct{ d time.Duration }

func (f *fakeIdle) IdleTime() (time.Duration, error) { return f.d, nil }

type errIdle struct{}

func (errIdle) IdleTime() (time.Duration, error) { return 0, errors.New("no display") }

type fakeInhibit struct {
	on     bool
	calls  []bool
	failOn bool
}

func (f *fakeInhibit) Acquire() error {
	f.calls = append(f.calls, true)
	if f.failOn {
		return errors.New("busy")
	}
	f.on = true
	return nil
}

func (f *fakeInhibit) Release() error {
	f.calls = append(f.calls, false)
	f.on = false
	return nil
}

type fakeNudge struct {
	jiggles []int
	shifts  int
	swaps   []struct {
		mod string
		n   int
	}
}

func (f *fakeNudge) Jiggle(px int) error {
	f.jiggles = append(f.jiggles, px)
	return nil
}
func (f *fakeNudge) TapShift() error {
	f.shifts++
	return nil
}
func (f *fakeNudge) SwitchWindows(mod string, n int) error {
	f.swaps = append(f.swaps, struct {
		mod string
		n   int
	}{mod, n})
	return nil
}

func baseSettings() Settings {
	return Settings{
		Interval: 10 * time.Second,
		Pixels:   1,
		Tabs:     1,
		Modifier: "alt",
		Inhibit:  true,
		Mouse:    true,
		Shift:    true,
		Switch:   false,
	}
}

func TestNoNudgeWhileActive(t *testing.T) {
	clock := &fakeClock{t: time.Unix(0, 0)}
	idle := &fakeIdle{d: 3 * time.Second}
	nudge := &fakeNudge{}
	inh := &fakeInhibit{}
	s := New(idle, nudge, inh, clock, nil, nil)
	s.Update(baseSettings())

	s.Tick()
	clock.t = clock.t.Add(30 * time.Second)
	s.Tick()

	if len(nudge.jiggles) != 0 || nudge.shifts != 0 {
		t.Fatalf("nudged while active: %+v shifts=%d", nudge.jiggles, nudge.shifts)
	}
	if !inh.on {
		t.Fatal("inhibitor should stay held while the user is active")
	}
}

func TestNudgeAfterIdle(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1000, 0)}
	idle := &fakeIdle{d: 10 * time.Second}
	nudge := &fakeNudge{}
	inh := &fakeInhibit{}
	var nudgedAt time.Time
	s := New(idle, nudge, inh, clock, nil, func(t time.Time) { nudgedAt = t })
	s.Update(baseSettings())

	s.Tick()
	if len(nudge.jiggles) != 1 || nudge.jiggles[0] != 1 || nudge.shifts != 1 {
		t.Fatalf("first nudge = jiggles %v shifts %d", nudge.jiggles, nudge.shifts)
	}
	if len(nudge.swaps) != 0 {
		t.Fatal("switch is off by default")
	}
	if !nudgedAt.Equal(clock.t) {
		t.Fatalf("nudge callback at %s", nudgedAt)
	}

	// Still idle, but the interval has not elapsed since the last nudge.
	clock.t = clock.t.Add(9 * time.Second)
	s.Tick()
	if nudge.shifts != 1 {
		t.Fatal("nudged again before the interval")
	}

	clock.t = clock.t.Add(1 * time.Second)
	s.Tick()
	if nudge.shifts != 2 || len(nudge.jiggles) != 2 {
		t.Fatalf("second nudge missing: jiggles %v shifts %d", nudge.jiggles, nudge.shifts)
	}
}

func TestPauseReleasesInhibitorAndSkipsInput(t *testing.T) {
	clock := &fakeClock{t: time.Unix(0, 0)}
	idle := &fakeIdle{d: time.Hour}
	nudge := &fakeNudge{}
	inh := &fakeInhibit{}
	s := New(idle, nudge, inh, clock, nil, nil)
	st := baseSettings()
	s.Update(st)
	s.Tick()
	if !inh.on || nudge.shifts != 1 {
		t.Fatal("expected a running nudge")
	}

	st.Paused = true
	s.Update(st)
	clock.t = clock.t.Add(time.Minute)
	s.Tick()
	if inh.on {
		t.Fatal("pause should release the inhibitor")
	}
	if nudge.shifts != 1 {
		t.Fatal("pause should not nudge")
	}

	st.Paused = false
	s.Update(st)
	clock.t = clock.t.Add(10 * time.Second)
	s.Tick()
	if !inh.on || nudge.shifts != 2 {
		t.Fatalf("resume held=%v shifts=%d", inh.on, nudge.shifts)
	}
}

func TestDurationAndPause(t *testing.T) {
	clock := &fakeClock{t: time.Unix(0, 0)}
	idle := &fakeIdle{d: 0}
	nudge := &fakeNudge{}
	inh := &fakeInhibit{}
	s := New(idle, nudge, inh, clock, nil, nil)
	st := baseSettings()
	st.Mouse, st.Shift = false, false
	st.Duration = time.Second
	s.Update(st)

	s.Tick() // t=0, starts the clock
	if inh.on != true {
		t.Fatal("inhibitor should be held")
	}
	clock.t = clock.t.Add(400 * time.Millisecond)
	s.Tick()
	if s.Expired() {
		t.Fatal("expired early")
	}

	st.Paused = true
	s.Update(st)
	clock.t = clock.t.Add(10 * time.Second)
	s.Tick()
	if s.Expired() {
		t.Fatal("paused time counted toward the limit")
	}
	if inh.on {
		t.Fatal("paused inhibitor still held")
	}

	st.Paused = false
	s.Update(st)
	clock.t = clock.t.Add(700 * time.Millisecond)
	s.Tick()
	if !s.Expired() {
		t.Fatal("expected expiry after the remaining active time")
	}
	if inh.on {
		t.Fatal("expiry should release the inhibitor")
	}
}

func TestDurationResetWhenChanged(t *testing.T) {
	clock := &fakeClock{t: time.Unix(0, 0)}
	idle := &fakeIdle{}
	s := New(idle, &fakeNudge{}, &fakeInhibit{}, clock, nil, nil)
	st := baseSettings()
	st.Mouse, st.Shift = false, false
	st.Duration = time.Second
	s.Update(st)
	s.Tick()
	clock.t = clock.t.Add(900 * time.Millisecond)
	s.Tick()

	st.Duration = 2 * time.Second
	s.Update(st)
	clock.t = clock.t.Add(1500 * time.Millisecond)
	s.Tick()
	if s.Expired() {
		t.Fatal("changing the limit should start it over")
	}
	clock.t = clock.t.Add(600 * time.Millisecond)
	s.Tick()
	if !s.Expired() {
		t.Fatal("expected expiry on the new limit")
	}
}

func TestSwitchTabsRandomWhenZero(t *testing.T) {
	clock := &fakeClock{t: time.Unix(0, 0)}
	idle := &fakeIdle{d: time.Minute}
	nudge := &fakeNudge{}
	s := New(idle, nudge, &fakeInhibit{}, clock, func(n int) int {
		if n != 3 {
			t.Fatalf("intn(%d)", n)
		}
		return 2
	}, nil)
	st := baseSettings()
	st.Mouse, st.Shift = false, false
	st.Switch = true
	st.Tabs = 0
	st.Modifier = "cmd"
	s.Update(st)
	s.Tick()
	if len(nudge.swaps) != 1 || nudge.swaps[0].n != 3 || nudge.swaps[0].mod != "cmd" {
		t.Fatalf("switch = %+v", nudge.swaps)
	}
}

func TestIdleErrorSkipsNudge(t *testing.T) {
	clock := &fakeClock{t: time.Unix(0, 0)}
	nudge := &fakeNudge{}
	inh := &fakeInhibit{}
	s := New(errIdle{}, nudge, inh, clock, nil, nil)
	s.Update(baseSettings())
	s.Tick()
	if len(nudge.jiggles) != 0 {
		t.Fatal("nudged without an idle reading")
	}
	if !inh.on {
		t.Fatal("inhibitor should still be held")
	}
}

func TestStopReleases(t *testing.T) {
	s := New(&fakeIdle{}, &fakeNudge{}, &fakeInhibit{}, &fakeClock{t: time.Unix(0, 0)}, nil, nil)
	st := baseSettings()
	st.Mouse, st.Shift = false, false
	s.Update(st)
	s.Tick()
	s.Stop()
	s.Stop()
	if s.held {
		t.Fatal("still held")
	}
}

func TestInhibitFailureDoesNotStick(t *testing.T) {
	inh := &fakeInhibit{failOn: true}
	s := New(&fakeIdle{d: 0}, &fakeNudge{}, inh, &fakeClock{t: time.Unix(0, 0)}, nil, nil)
	st := baseSettings()
	st.Mouse, st.Shift = false, false
	s.Update(st)
	s.Tick()
	if s.held {
		t.Fatal("failed acquire marked as held")
	}
	inh.failOn = false
	s.Tick()
	if !inh.on {
		t.Fatal("retry should acquire")
	}
}
