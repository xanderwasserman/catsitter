// Package tray puts Catsitter in the menu bar or system tray.
package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"sync"
	"time"

	"fyne.io/systray"
	"github.com/xanderwasserman/catsitter/internal/control"
)

// Handle starts the tray on the UI thread and stops it on shutdown.
type Handle struct {
	ctrl  control.Controller
	start func()
	end   func()
	once  sync.Once
}

// New builds the tray. Start must run on the main thread.
func New(ctrl control.Controller) *Handle {
	h := &Handle{ctrl: ctrl}
	start, end := systray.RunWithExternalLoop(func() { h.ready() }, func() {})
	h.start = start
	h.end = end
	return h
}

// Start shows the icon. It is safe to call more than once.
func (h *Handle) Start() {
	if h == nil || h.start == nil {
		return
	}
	h.once.Do(h.start)
}

// Stop removes the icon.
func (h *Handle) Stop() {
	if h == nil || h.end == nil {
		return
	}
	h.end()
	h.end = nil
}

func (h *Handle) ready() {
	systray.SetIcon(iconPNG())
	systray.SetTooltip("Catsitter")
	pause := systray.AddMenuItem("Pause", "Pause or resume")
	systray.AddSeparator()
	inhibit := systray.AddMenuItemCheckbox("Keep awake", "Sleep inhibitor", true)
	mouse := systray.AddMenuItemCheckbox("Mouse jiggle", "Nudge the pointer", true)
	shift := systray.AddMenuItemCheckbox("Shift tap", "Tap Shift", true)
	sw := systray.AddMenuItemCheckbox("Switch windows", "Alt or Cmd+Tab", false)
	systray.AddSeparator()
	interval := systray.AddMenuItem("Idle every", "How long to wait")
	var intervalItems []*systray.MenuItem
	for _, d := range control.Intervals {
		intervalItems = append(intervalItems, interval.AddSubMenuItemCheckbox(control.IntervalLabel(d), "", false))
	}
	duration := systray.AddMenuItem("For", "How long to keep watch")
	var durationItems []*systray.MenuItem
	for _, d := range control.Durations {
		durationItems = append(durationItems, duration.AddSubMenuItemCheckbox(control.DurationLabel(d), "", false))
	}
	systray.AddSeparator()
	hide := systray.AddMenuItem("Hide cat", "Hide the cat and keep working")
	systray.AddSeparator()
	quit := systray.AddMenuItem("Quit", "Quit Catsitter")

	go func() {
		sync := time.NewTicker(400 * time.Millisecond)
		defer sync.Stop()
		for {
			select {
			case <-pause.ClickedCh:
				h.ctrl.TogglePause()
			case <-inhibit.ClickedCh:
				h.ctrl.SetInhibit(!h.ctrl.View().Inhibit)
			case <-mouse.ClickedCh:
				h.ctrl.SetMouse(!h.ctrl.View().Mouse)
			case <-shift.ClickedCh:
				h.ctrl.SetShift(!h.ctrl.View().Shift)
			case <-sw.ClickedCh:
				h.ctrl.SetSwitch(!h.ctrl.View().Switch)
			case <-hide.ClickedCh:
				h.ctrl.ToggleHidden()
			case <-quit.ClickedCh:
				h.ctrl.Quit()
				return
			case <-sync.C:
				view := h.ctrl.View()
				if view.Quitting {
					return
				}
				if view.Paused {
					pause.SetTitle("Resume")
				} else {
					pause.SetTitle("Pause")
				}
				setCheck(inhibit, view.Inhibit)
				setCheck(mouse, view.Mouse)
				setCheck(shift, view.Shift)
				setCheck(sw, view.Switch)
				if view.Hidden {
					hide.SetTitle("Show cat")
				} else {
					hide.SetTitle("Hide cat")
				}
				interval.SetTitle("Idle every " + control.IntervalLabel(view.Interval))
				for i, item := range intervalItems {
					setCheck(item, control.Intervals[i] == view.Interval)
				}
				duration.SetTitle("For " + control.DurationLabel(view.Duration))
				for i, item := range durationItems {
					setCheck(item, control.Durations[i] == view.Duration)
				}
			}
		}
	}()
	for i, item := range intervalItems {
		i, item := i, item
		go func() {
			for range item.ClickedCh {
				h.ctrl.SetInterval(control.Intervals[i])
			}
		}()
	}
	for i, item := range durationItems {
		i, item := i, item
		go func() {
			for range item.ClickedCh {
				h.ctrl.SetDuration(control.Durations[i])
			}
		}()
	}
}

func setCheck(item *systray.MenuItem, on bool) {
	if on {
		item.Check()
	} else {
		item.Uncheck()
	}
}

func iconPNG() []byte {
	const n = 16
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	// A tiny original cat face for the tray.
	rows := []string{
		"....KKKKKK....",
		"..KKOOOOOOKK..",
		".KPPPOOOOPPPK.",
		"KOOOEEBOEEBOOK",
		"KOOOEBBOEBBOOK",
		"KOOOOONNOOOOOK",
		".KOOOWWWWWOOK.",
		"..KKOOOOOOKK..",
		".KOOOLLLLOOOK.",
		"KOOOLLLLLLOOOK",
		"KOODLLLLLLDOOK",
		".KOOODDDDOOOK.",
		"..KKK....KKK..",
		"..............",
		"..............",
		"..............",
	}
	colors := map[byte]color.RGBA{
		'K': {42, 32, 28, 255},
		'O': {232, 138, 58, 255},
		'P': {242, 168, 184, 255},
		'E': {54, 120, 72, 255},
		'B': {28, 22, 26, 255},
		'N': {214, 96, 122, 255},
		'W': {255, 246, 234, 255},
		'L': {246, 196, 138, 255},
		'D': {176, 88, 36, 255},
	}
	for y, row := range rows {
		pad := 0
		if len(row) < n {
			pad = (n - len(row)) / 2
		}
		for x := 0; x < len(row) && pad+x < n; x++ {
			if c, ok := colors[row[x]]; ok {
				img.SetRGBA(pad+x, y, c)
			}
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
