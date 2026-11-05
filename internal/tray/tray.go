// Package tray puts Catsitter in the menu bar or system tray.
package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"log"
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

// New builds the tray. New and Start must run on the main thread.
// On Windows the tray window belongs to the thread that calls New.
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
	h.once.Do(func() {
		defer survive("start")
		h.start()
	})
}

// survive keeps a Go panic in the tray from ending the app. The cat menu still works.
func survive(step string) {
	if r := recover(); r != nil {
		log.Printf("tray %s failed: %v", step, r)
	}
}

// Stop removes the icon where it would outlive the process. Call it on the
// main thread after the pet window closes.
func (h *Handle) Stop() {
	if !removeOnStop || h == nil || h.end == nil {
		return
	}
	end := h.end
	h.end = nil
	defer survive("stop")
	end()
}

func (h *Handle) ready() {
	defer survive("setup")
	systray.SetIcon(menuBarIcon())
	systray.SetTooltip("Catsitter")
	paused := systray.AddMenuItem("Nudges paused: Catsitter needs Accessibility", "macOS Accessibility permission")
	paused.Disable()
	paused.Hide()
	allow := systray.AddMenuItem("Allow Accessibility…", "Open the macOS permission settings")
	allow.Hide()
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
			case <-allow.ClickedCh:
				h.ctrl.RequestAccess()
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
				if view.Trusted {
					paused.Hide()
					allow.Hide()
					systray.SetTooltip("Catsitter")
				} else {
					paused.Show()
					allow.Show()
					systray.SetTooltip("Catsitter: nudges paused until you allow Accessibility")
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
				duration.SetTitle(control.WatchLabel(view.Duration))
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

// iconPNG is iconImage encoded as PNG.
func iconPNG(scale int, rim bool) []byte {
	var buf bytes.Buffer
	_ = png.Encode(&buf, iconImage(scale, rim))
	return buf.Bytes()
}

// iconImage draws the tray cat face on a 16-unit grid, scale pixels per unit.
// rim adds a light edge around the dark outline and moves the face down
// one unit to make room for it.
func iconImage(scale int, rim bool) *image.RGBA {
	const n = 16
	grid := image.NewRGBA(image.Rect(0, 0, n, n))
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
	top := 0
	if rim {
		top = 1
	}
	for y, row := range rows {
		pad := (n - len(row)) / 2
		for x := 0; x < len(row); x++ {
			if c, ok := colors[row[x]]; ok {
				grid.SetRGBA(pad+x, top+y, c)
			}
		}
	}
	if rim {
		addRim(grid, color.RGBA{128, 123, 117, 128})
	}
	img := image.NewRGBA(image.Rect(0, 0, n*scale, n*scale))
	for y := 0; y < n*scale; y++ {
		for x := 0; x < n*scale; x++ {
			img.SetRGBA(x, y, grid.RGBAAt(x/scale, y/scale))
		}
	}
	return img
}

// addRim fills each empty cell that touches the art, including diagonally.
func addRim(grid *image.RGBA, c color.RGBA) {
	b := grid.Bounds()
	var edge []image.Point
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if grid.RGBAAt(x, y).A == 0 && touchesArt(grid, x, y) {
				edge = append(edge, image.Pt(x, y))
			}
		}
	}
	for _, p := range edge {
		grid.SetRGBA(p.X, p.Y, c)
	}
}

func touchesArt(grid *image.RGBA, x, y int) bool {
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			p := image.Pt(x+dx, y+dy)
			if p.In(grid.Bounds()) && grid.RGBAAt(p.X, p.Y).A == 255 {
				return true
			}
		}
	}
	return false
}
