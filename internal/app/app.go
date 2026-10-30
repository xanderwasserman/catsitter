// Package app runs the scheduler and keeps the config, tray, and pet in sync.
package app

import (
	"context"
	"log"
	"math/rand/v2"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xanderwasserman/catsitter/internal/config"
	"github.com/xanderwasserman/catsitter/internal/control"
	"github.com/xanderwasserman/catsitter/internal/host"
	"github.com/xanderwasserman/catsitter/internal/platform"
	"github.com/xanderwasserman/catsitter/internal/scheduler"
)

// App is the running Catsitter session.
type App struct {
	mu        sync.Mutex
	cfg       config.Config
	path      string
	paused    bool
	sched     *scheduler.Scheduler
	gate      platform.Gate
	closeFn   func()
	cancel    context.CancelFunc
	quit      atomic.Bool
	trusted   atomic.Bool
	lastNudge atomic.Int64
	prompted  atomic.Bool
}

// New prepares a session. Call Start to run the scheduler.
func New(cfg config.Config, path string, b host.Backends) *App {
	a := &App{
		cfg:     cfg,
		path:    path,
		gate:    b.Gate,
		closeFn: b.Close,
	}
	a.trusted.Store(b.Gate == nil || b.Gate.Trusted())
	a.sched = scheduler.New(b.Idle, b.Nudge, b.Inhibit, nil, rand.IntN, func(t time.Time) {
		a.lastNudge.Store(t.UnixNano())
	})
	a.push()
	return a
}

// Start runs the scheduler until Quit or the duration limit.
func (a *App) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	a.mu.Lock()
	a.cancel = cancel
	a.mu.Unlock()
	go a.signals()
	go a.watchPermission(ctx)
	go a.loop(ctx)
}

func (a *App) loop(ctx context.Context) {
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	defer a.sched.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			a.sched.Tick()
			if a.sched.Expired() {
				log.Printf("the watch is over")
				a.Quit()
				return
			}
		}
	}
}

func (a *App) signals() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, listenSignals()...)
	<-ch
	log.Printf("stopping")
	a.Quit()
}

func (a *App) watchPermission(ctx context.Context) {
	if a.gate == nil {
		return
	}
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	a.applyTrust(a.gate.Trusted())
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			a.applyTrust(a.gate.Trusted())
		}
	}
}

func (a *App) applyTrust(ok bool) {
	a.trusted.Store(ok)
	a.mu.Lock()
	want := a.cfg.Mouse || a.cfg.Shift || a.cfg.Switch
	a.pushLocked()
	a.mu.Unlock()
	if ok || !want || a.prompted.Swap(true) {
		return
	}
	log.Printf("accessibility permission is missing; nudges wait, the sleep inhibitor still runs")
	a.gate.Prompt()
}

// Shutdown releases the inhibitor and writes the config.
func (a *App) Shutdown() {
	a.Quit()
	if a.sched != nil {
		a.sched.Stop()
	}
	a.mu.Lock()
	cfg := a.cfg
	path := a.path
	a.mu.Unlock()
	if path != "" {
		if err := config.Save(path, cfg); err != nil {
			log.Printf("save config: %v", err)
		}
	}
	if a.closeFn != nil {
		a.closeFn()
		a.closeFn = nil
	}
}

func (a *App) View() control.View {
	a.mu.Lock()
	defer a.mu.Unlock()
	var last time.Time
	if n := a.lastNudge.Load(); n != 0 {
		last = time.Unix(0, n)
	}
	return control.View{
		Paused:    a.paused,
		Inhibit:   a.cfg.Inhibit,
		Mouse:     a.cfg.Mouse,
		Shift:     a.cfg.Shift,
		Switch:    a.cfg.Switch,
		Interval:  a.cfg.IntervalDuration(),
		Duration:  a.cfg.LimitDuration(),
		Hidden:    a.cfg.HidePet,
		LastNudge: last,
		WindowX:   a.cfg.WindowX,
		WindowY:   a.cfg.WindowY,
		WindowSet: a.cfg.WindowSet,
		Trusted:   a.trusted.Load(),
		Quitting:  a.quit.Load(),
	}
}

func (a *App) TogglePause() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.paused = !a.paused
	a.pushLocked()
	if a.paused {
		log.Printf("paused")
	} else {
		log.Printf("resumed")
	}
}

func (a *App) SetInhibit(v bool) { a.setBool(func(c *config.Config) { c.Inhibit = v }) }
func (a *App) SetMouse(v bool)   { a.setBool(func(c *config.Config) { c.Mouse = v }) }
func (a *App) SetShift(v bool)   { a.setBool(func(c *config.Config) { c.Shift = v }) }
func (a *App) SetSwitch(v bool)  { a.setBool(func(c *config.Config) { c.Switch = v }) }

func (a *App) SetInterval(d time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.Interval = config.FormatDuration(d)
	if a.cfg.Interval == "" {
		a.cfg.Interval = "10s"
	}
	a.pushLocked()
	a.saveLocked()
}

func (a *App) SetDuration(d time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.Duration = config.FormatDuration(d)
	a.pushLocked()
	a.saveLocked()
}

func (a *App) ToggleHidden() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cfg.HidePet = !a.cfg.HidePet
	a.saveLocked()
}

func (a *App) Quit() {
	a.quit.Store(true)
	a.mu.Lock()
	cancel := a.cancel
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (a *App) SaveWindow(x, y int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cfg.WindowSet && a.cfg.WindowX == x && a.cfg.WindowY == y {
		return
	}
	a.cfg.WindowX = x
	a.cfg.WindowY = y
	a.cfg.WindowSet = true
	a.saveLocked()
}

func (a *App) setBool(fn func(*config.Config)) {
	a.mu.Lock()
	defer a.mu.Unlock()
	fn(&a.cfg)
	a.pushLocked()
	a.saveLocked()
}

func (a *App) push() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.pushLocked()
}

func (a *App) pushLocked() {
	st := scheduler.Settings{
		Interval: a.cfg.IntervalDuration(),
		Pixels:   a.cfg.Pixels,
		Tabs:     a.cfg.Tabs,
		Modifier: a.cfg.ModifierKey(),
		Inhibit:  a.cfg.Inhibit,
		Mouse:    a.cfg.Mouse,
		Shift:    a.cfg.Shift,
		Switch:   a.cfg.Switch,
		Duration: a.cfg.LimitDuration(),
		Paused:   a.paused,
	}
	if !a.trusted.Load() {
		st.Mouse, st.Shift, st.Switch = false, false, false
	}
	a.sched.Update(st)
}

func (a *App) saveLocked() {
	if a.path == "" {
		return
	}
	if err := config.Save(a.path, a.cfg); err != nil {
		log.Printf("save config: %v", err)
	}
}
