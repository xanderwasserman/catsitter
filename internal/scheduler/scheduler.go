// Package scheduler decides when to nudge an idle machine.
// It never posts input while the user is active. The sleep inhibitor,
// when enabled, stays held for the whole running session.
package scheduler

import (
	"log"
	"sync"
	"time"

	"github.com/xanderwasserman/catsitter/internal/platform"
)

// Settings is a snapshot of the toggles the scheduler honors.
type Settings struct {
	Interval time.Duration
	Pixels   int
	Tabs     int
	Modifier string
	Inhibit  bool
	Mouse    bool
	Shift    bool
	Switch   bool
	Duration time.Duration // zero means unlimited
	Paused   bool
}

// Clock supplies the current time. Tests pass a fake.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// Scheduler ticks from the outside so tests can drive it without sleeping.
type Scheduler struct {
	idle    platform.IdleDetector
	nudge   platform.Nudger
	inhibit platform.Inhibitor
	clock   Clock
	intn    func(n int) int
	onNudge func(time.Time)

	mu          sync.Mutex
	settings    Settings
	lastTick    time.Time
	used        time.Duration
	lastNudge   time.Time
	held        bool
	expired     bool
	idleWarn    bool
	inhibitWarn bool
}

// New builds a scheduler. intn chooses a random tab count when Tabs is 0
// (it should return a value in [0, n)). onNudge, when set, runs after a
// nudge and must not call back into the scheduler.
func New(idle platform.IdleDetector, nudge platform.Nudger, inhibit platform.Inhibitor, clock Clock, intn func(n int) int, onNudge func(time.Time)) *Scheduler {
	if clock == nil {
		clock = realClock{}
	}
	return &Scheduler{
		idle:    idle,
		nudge:   nudge,
		inhibit: inhibit,
		clock:   clock,
		intn:    intn,
		onNudge: onNudge,
	}
}

// Update replaces the settings used on the next tick.
// Changing Duration starts the limit over.
func (s *Scheduler) Update(next Settings) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if next.Duration != s.settings.Duration {
		s.used = 0
	}
	s.settings = next
}

// Expired reports whether the duration limit has been reached.
func (s *Scheduler) Expired() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.expired
}

// Tick runs one pass. Call it from a single goroutine.
func (s *Scheduler) Tick() {
	s.mu.Lock()
	now := s.clock.Now()
	var dt time.Duration
	if !s.lastTick.IsZero() {
		dt = now.Sub(s.lastTick)
		if dt < 0 {
			dt = 0
		}
	}
	s.lastTick = now
	st := s.settings

	if st.Paused {
		s.setInhibit(false)
		s.mu.Unlock()
		return
	}
	if st.Duration > 0 {
		s.used += dt
		if s.used >= st.Duration {
			s.expired = true
			s.setInhibit(false)
			s.mu.Unlock()
			return
		}
	}
	s.setInhibit(st.Inhibit)

	if !st.Mouse && !st.Shift && !st.Switch {
		s.mu.Unlock()
		return
	}
	idleFor, err := s.idle.IdleTime()
	if err != nil {
		if !s.idleWarn {
			s.idleWarn = true
			log.Printf("idle time unavailable: %v", err)
		}
		s.mu.Unlock()
		return
	}
	interval := st.Interval
	if interval < time.Second {
		interval = time.Second
	}
	if idleFor < interval {
		s.mu.Unlock()
		return
	}
	if !s.lastNudge.IsZero() && now.Sub(s.lastNudge) < interval {
		s.mu.Unlock()
		return
	}
	s.lastNudge = now
	pixels := st.Pixels
	if pixels < 1 {
		pixels = 1
	}
	tabs := st.Tabs
	if tabs <= 0 {
		tabs = 1
		if s.intn != nil {
			tabs = s.intn(3) + 1
		}
	}
	mod := st.Modifier
	doMouse, doShift, doSwitch := st.Mouse, st.Shift, st.Switch
	s.mu.Unlock()

	if doMouse {
		if err := s.nudge.Jiggle(pixels); err != nil {
			log.Printf("mouse jiggle: %v", err)
		} else {
			log.Printf("mouse jiggle (%dpx)", pixels)
		}
	}
	if doShift {
		if err := s.nudge.TapShift(); err != nil {
			log.Printf("shift tap: %v", err)
		} else {
			log.Printf("shift tap")
		}
	}
	if doSwitch {
		if err := s.nudge.SwitchWindows(mod, tabs); err != nil {
			log.Printf("window switch: %v", err)
		} else {
			log.Printf("window switch (%s+%d)", mod, tabs)
		}
	}
	if s.onNudge != nil {
		s.onNudge(now)
	}
}

// Stop releases the inhibitor. It is safe to call more than once.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setInhibit(false)
}

// setInhibit runs while s.mu is held.
func (s *Scheduler) setInhibit(on bool) {
	if s.inhibit == nil || on == s.held {
		return
	}
	var err error
	if on {
		err = s.inhibit.Acquire()
	} else {
		err = s.inhibit.Release()
	}
	if err != nil {
		if on && !s.inhibitWarn {
			s.inhibitWarn = true
			log.Printf("sleep inhibitor: %v", err)
		}
		if on {
			return
		}
	}
	s.inhibitWarn = false
	s.held = on
}
