package pet

import (
	"image/color"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"github.com/xanderwasserman/catsitter/internal/control"
	"github.com/xanderwasserman/catsitter/internal/platform"
	"github.com/xanderwasserman/catsitter/internal/tray"
)

const (
	menuW   = 196
	rowH    = 20
	menuPad = 8
)

// Run shows the pet until the controller asks to quit.
// The tray is started on the first frame so it shares the UI thread.
func Run(p Pet, ctrl control.Controller, bar *tray.Handle, pump func(), overlay func() error) error {
	catW, catH := p.Size()
	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowFloating(true)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	ebiten.SetWindowTitle(platform.OverlayTitle)
	ebiten.SetWindowSize(catW, catH)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetTPS(12)
	op := &ebiten.RunGameOptions{
		ScreenTransparent: true,
		SkipTaskbar:       true,
		X11ClassName:      platform.OverlayTitle,
		X11InstanceName:   "catsitter",
	}
	return ebiten.RunGameWithOptions(&game{
		pet:     p,
		ctrl:    ctrl,
		tray:    bar,
		catW:    catW,
		catH:    catH,
		pixel:   ebiten.NewImage(1, 1),
		pump:    pump,
		overlay: overlay,
	}, op)
}

type game struct {
	pet  Pet
	ctrl control.Controller
	tray *tray.Handle
	catW int
	catH int

	pixel   *ebiten.Image
	pump    func()
	overlay func() error

	overlayDone      bool
	overlayWarned    bool
	overlayTries     int
	menuSawFocus     bool
	placed           bool
	menu             bool
	hidden           bool
	savedX, savedY   int
	dragging         bool
	dragStartWindowX int
	dragStartWindowY int
	dragStartCursorX int
	dragStartCursorY int
	cursorToWindowX  float64
	cursorToWindowY  float64
	dragged          bool
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	w, h := g.screenSize()
	if w > 0 && h > 0 {
		g.cursorToWindowX = float64(outsideWidth) / float64(w)
		g.cursorToWindowY = float64(outsideHeight) / float64(h)
	}
	return w, h
}

func (g *game) screenSize() (int, int) {
	if g.menu {
		h := g.catH
		need := menuPad*2 + len(menuRows(g.ctrl.View()))*rowH
		if need > h {
			h = need
		}
		return g.catW + menuW, h
	}
	return g.catW, g.catH
}

func (g *game) Update() error {
	if g.pump != nil {
		g.pump()
	}
	g.keepOutOfSwitcher()
	// CloseMainWindow and the window's close button both arrive here.
	// Quit first so Shutdown can save the config before the process ends.
	if ebiten.IsWindowBeingClosed() {
		g.ctrl.Quit()
		return ebiten.Termination
	}
	view := g.ctrl.View()
	if view.Quitting {
		return ebiten.Termination
	}
	if !g.placed {
		g.place(view)
		if g.tray != nil {
			g.tray.Start()
		}
		g.placed = true
	}
	g.syncHidden(view)
	if g.hidden {
		ebiten.SetTPS(2)
		return nil
	}
	ebiten.SetTPS(12)
	g.syncWindowSize()
	g.pointer(view)
	return nil
}

func (g *game) place(view control.View) {
	if view.WindowSet {
		ebiten.SetWindowPosition(view.WindowX, view.WindowY)
		return
	}
	mon := ebiten.Monitor()
	if mon == nil {
		return
	}
	sw, sh := mon.Size()
	x, y := sw-g.catW-80, sh-g.catH-140
	if x < 16 {
		x = 16
	}
	if y < 16 {
		y = 16
	}
	ebiten.SetWindowPosition(x, y)
	g.ctrl.SaveWindow(x, y)
}

func (g *game) syncHidden(view control.View) {
	if view.Hidden == g.hidden {
		return
	}
	if view.Hidden {
		g.savedX, g.savedY = ebiten.WindowPosition()
		g.menu = false
		g.dragging = false
		ebiten.SetWindowPosition(-3000, -3000)
		g.hidden = true
		return
	}
	ebiten.SetWindowPosition(g.savedX, g.savedY)
	g.hidden = false
}

func (g *game) syncWindowSize() {
	w, h := g.screenSize()
	ww, wh := ebiten.WindowSize()
	if ww != w || wh != h {
		ebiten.SetWindowSize(w, h)
	}
}

// keepOutOfSwitcher applies the platform hint on every frame. Resizing the
// window replaces the macOS collection behavior, so one application is not enough.
func (g *game) keepOutOfSwitcher() {
	if g.overlay == nil || g.overlayDone {
		return
	}
	if err := g.overlay(); err != nil {
		if !g.overlayWarned {
			log.Printf("window switcher: %v", err)
			g.overlayWarned = true
		}
		g.overlayTries++
		if g.overlayTries >= 60 {
			g.overlayDone = true
		}
		return
	}
	g.overlayTries = 0
	g.overlayWarned = false
}

func (g *game) pointer(view control.View) {
	cx, cy := ebiten.CursorPosition()
	open, saw, activate, consumed := nextMenu(g.menu, menuEvent{
		Right:    inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
		Left:     inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		Escape:   inpututil.IsKeyJustPressed(ebiten.KeyEscape),
		Focused:  ebiten.IsFocused(),
		SawFocus: g.menuSawFocus,
		OnMenu:   cx >= g.catW,
	})
	if activate {
		g.activate(cy, view)
	}
	g.menu = open
	g.menuSawFocus = saw
	if consumed {
		g.dragging = false
		return
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if g.dragging && g.dragged {
			x, y := ebiten.WindowPosition()
			g.ctrl.SaveWindow(x, y)
		}
		g.dragging = false
		g.dragged = false
		return
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.dragging = true
		g.dragged = false
		g.dragStartWindowX, g.dragStartWindowY = ebiten.WindowPosition()
		g.dragStartCursorX, g.dragStartCursorY = cx, cy
		return
	}
	if !g.dragging {
		return
	}
	dx := int(float64(cx-g.dragStartCursorX) * g.cursorToWindowX)
	dy := int(float64(cy-g.dragStartCursorY) * g.cursorToWindowY)
	if dx != 0 || dy != 0 {
		g.dragged = true
	}
	wx, wy := ebiten.WindowPosition()
	ebiten.SetWindowPosition(wx+dx, wy+dy)
}

func (g *game) activate(cy int, view control.View) {
	rows := menuRows(view)
	idx := (cy - menuPad) / rowH
	if idx < 0 || idx >= len(rows) {
		return
	}
	switch rows[idx].id {
	case "pause":
		g.ctrl.TogglePause()
	case "inhibit":
		g.ctrl.SetInhibit(!view.Inhibit)
	case "mouse":
		g.ctrl.SetMouse(!view.Mouse)
	case "shift":
		g.ctrl.SetShift(!view.Shift)
	case "switch":
		g.ctrl.SetSwitch(!view.Switch)
	case "interval":
		g.ctrl.SetInterval(control.NextInterval(view.Interval))
	case "duration":
		g.ctrl.SetDuration(control.NextDuration(view.Duration))
	case "hide":
		g.ctrl.ToggleHidden()
	case "quit":
		g.ctrl.Quit()
	}
}

func (g *game) Draw(screen *ebiten.Image) {
	view := g.ctrl.View()
	if view.Hidden {
		return
	}
	act := activity(view, ebiten.Tick())
	frame := g.pet.Frame(act, ebiten.Tick())
	if frame != nil {
		op := &ebiten.DrawImageOptions{}
		fw := float64(frame.Bounds().Dx())
		if fw > 0 {
			op.GeoM.Scale(float64(g.catW)/fw, float64(g.catH)/fw)
		}
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(frame, op)
	}
	if !g.menu {
		return
	}
	_, h := g.screenSize()
	g.pixel.Fill(color.White)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(menuW), float64(h))
	op.GeoM.Translate(float64(g.catW), 0)
	op.ColorScale.ScaleWithColor(color.RGBA{42, 33, 28, 235})
	screen.DrawImage(g.pixel, op)
	rows := menuRows(view)
	ink := color.RGBA{255, 246, 234, 255}
	for i, row := range rows {
		text.Draw(screen, row.label, basicfont.Face7x13, g.catW+12, menuPad+i*rowH+14, ink)
	}
}

func activity(view control.View, tick int64) Activity {
	if view.Paused {
		return ActivitySleep
	}
	if !view.LastNudge.IsZero() && time.Since(view.LastNudge) < 900*time.Millisecond {
		return ActivityPaws
	}
	switch (tick / 12 / 3) % 5 {
	case 0, 1:
		return ActivityIdle
	case 2:
		return ActivitySit
	case 3:
		return ActivityWalk
	default:
		return ActivitySit
	}
}

type menuRow struct {
	id    string
	label string
}

func menuRows(view control.View) []menuRow {
	pause := "Pause"
	if view.Paused {
		pause = "Resume"
	}
	hide := "Hide cat"
	if view.Hidden {
		hide = "Show cat"
	}
	mark := func(on bool) string {
		if on {
			return "[x] "
		}
		return "[ ] "
	}
	rows := []menuRow{
		{id: "pause", label: pause},
		{id: "inhibit", label: mark(view.Inhibit) + "Keep awake"},
		{id: "mouse", label: mark(view.Mouse) + "Mouse jiggle"},
		{id: "shift", label: mark(view.Shift) + "Shift tap"},
		{id: "switch", label: mark(view.Switch) + "Switch windows"},
		{id: "interval", label: "Idle every " + control.IntervalLabel(view.Interval)},
		{id: "duration", label: control.WatchLabel(view.Duration)},
		{id: "hide", label: hide},
		{id: "quit", label: "Quit"},
	}
	if view.Trusted {
		return rows
	}
	return append([]menuRow{{id: "access", label: "Accessibility is off"}}, rows...)
}
