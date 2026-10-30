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

// Desktop is the platform glue the pet window needs.
type Desktop struct {
	Pump    func()
	Overlay func() error
	// MenuGrab takes keyboard and pointer input while the menu is open and
	// reports a press outside the window or Escape. Nil where the window
	// already receives those events.
	MenuGrab func(open bool) bool
	// Pointer returns the pointer in window-position units.
	// deviceScale is the monitor scale. ok is false when it is unavailable.
	Pointer func(deviceScale float64) (x, y int, ok bool)
	// Press reports a button-press position and whether that button is held.
	// Nil where the window's own click is the press position.
	Press func(deviceScale float64) (x, y int, pressed, down bool)
	// NoteFocus records the app that should receive keys after the menu.
	// RestoreFocus hands them back when this process still holds the keyboard.
	// Both are nil where the menu does not take focus.
	NoteFocus    func()
	RestoreFocus func()
}

// Run shows the pet until the controller asks to quit.
// The tray is started on the first frame so it shares the UI thread.
func Run(p Pet, ctrl control.Controller, bar *tray.Handle, desk Desktop) error {
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
		pet:          p,
		ctrl:         ctrl,
		tray:         bar,
		catW:         catW,
		catH:         catH,
		pixel:        ebiten.NewImage(1, 1),
		pump:         desk.Pump,
		overlay:      desk.Overlay,
		menuGrab:     desk.MenuGrab,
		pointAt:      desk.Pointer,
		buttonAt:     desk.Press,
		noteFocus:    desk.NoteFocus,
		restoreFocus: desk.RestoreFocus,
	}, op)
}

type game struct {
	pet  Pet
	ctrl control.Controller
	tray *tray.Handle
	catW int
	catH int

	pixel        *ebiten.Image
	pump         func()
	overlay      func() error
	menuGrab     func(open bool) bool
	pointAt      func(deviceScale float64) (x, y int, ok bool)
	buttonAt     func(deviceScale float64) (x, y int, pressed, down bool)
	noteFocus    func()
	restoreFocus func()

	overlayDone    bool
	overlayWarned  bool
	overlayTries   int
	menuSawFocus   bool
	placed         bool
	menu           bool
	hidden         bool
	haveCat        bool
	catX, catY     int
	savedX, savedY int
	lay            placement
	dragging       bool
	dragged        bool
	dragAbs        bool
	dragCatX       int
	dragCatY       int
	dragPtrX       int
	dragPtrY       int
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenSize()
}

func (g *game) screenSize() (int, int) {
	if g.menu && g.lay.Window.W > 0 && g.lay.Window.H > 0 {
		return g.lay.Window.W, g.lay.Window.H
	}
	return g.catW, g.catH
}

func (g *game) Update() error {
	if g.noteFocus != nil {
		g.noteFocus()
	}
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
	wasMenu := g.menu
	g.syncHidden(view)
	if g.hidden {
		g.syncMenuGrab()
		if wasMenu {
			g.handBack()
		}
		ebiten.SetTPS(2)
		return nil
	}
	if g.menu {
		ebiten.SetTPS(60)
	} else {
		ebiten.SetTPS(12)
	}
	g.pointer(view)
	g.applyWindow(monitorSize())
	return nil
}

func (g *game) place(view control.View) {
	if view.WindowSet {
		g.catX, g.catY = view.WindowX, view.WindowY
		g.haveCat = true
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
	g.catX, g.catY = x, y
	g.haveCat = true
	ebiten.SetWindowPosition(x, y)
	g.ctrl.SaveWindow(x, y)
}

func (g *game) syncHidden(view control.View) {
	if view.Hidden == g.hidden {
		return
	}
	if view.Hidden {
		g.savedX, g.savedY = g.catX, g.catY
		g.menu = false
		g.dragging = false
		ebiten.SetWindowPosition(-3000, -3000)
		g.hidden = true
		return
	}
	g.catX, g.catY = g.savedX, g.savedY
	g.haveCat = true
	ebiten.SetWindowPosition(g.savedX, g.savedY)
	g.hidden = false
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
	outside := false
	if g.menu && g.menuGrab != nil && g.menuGrab(true) {
		outside = true
	}
	open, saw, activate, consumed := nextMenu(g.menu, menuEvent{
		Right:    inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
		Left:     inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		Escape:   inpututil.IsKeyJustPressed(ebiten.KeyEscape),
		Outside:  outside,
		Focused:  ebiten.IsFocused(),
		SawFocus: g.menuSawFocus,
		OnMenu:   g.lay.Menu.contains(cx, cy),
	})
	if activate {
		g.activate(cy-g.lay.Menu.Y, view)
	}
	closing := g.menu && !open
	g.menu = open
	g.menuSawFocus = saw
	g.syncMenuGrab()
	if closing {
		g.handBack()
	}
	px, py, pressed, held := g.readPress()
	if g.menu || closing {
		pressed = false
		held = false
	}
	if consumed {
		g.dragging = false
		return
	}
	if !g.haveCat {
		return
	}
	just := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	if !g.dragging && (pressed || just) {
		sx, sy, sampled := g.samplePointer()
		ax, ay, abs := dragAnchor(px, py, pressed, sampled, sx, sy, cx, cy)
		g.dragging = true
		g.dragged = false
		g.dragAbs = abs
		g.dragCatX, g.dragCatY = g.catX, g.catY
		g.dragPtrX, g.dragPtrY = ax, ay
	}
	down := ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) || held
	if g.dragging {
		g.moveDrag(cx, cy)
	}
	if !down {
		if g.dragging && g.dragged {
			g.ctrl.SaveWindow(g.catX, g.catY)
		}
		g.dragging = false
		g.dragged = false
	}
}

func (g *game) handBack() {
	if g.restoreFocus != nil {
		g.restoreFocus()
	}
}

func (g *game) readPress() (x, y int, pressed, down bool) {
	if g.buttonAt == nil {
		return 0, 0, false, false
	}
	return g.buttonAt(g.deviceScale())
}

func (g *game) moveDrag(cx, cy int) {
	var nx, ny int
	if g.dragAbs {
		x, y, ok := g.samplePointer()
		if !ok {
			return
		}
		nx, ny = draggedOrigin(g.dragCatX, g.dragCatY, g.dragPtrX, g.dragPtrY, x, y)
	} else {
		// The cursor is window-local. The press-time offset stays under it.
		wx, wy := ebiten.WindowPosition()
		nx = wx + cx - g.dragPtrX
		ny = wy + cy - g.dragPtrY
	}
	if nx != g.catX || ny != g.catY {
		g.dragged = true
	}
	g.catX, g.catY = nx, ny
}

func (g *game) deviceScale() float64 {
	if mon := ebiten.Monitor(); mon != nil && mon.DeviceScaleFactor() > 0 {
		return mon.DeviceScaleFactor()
	}
	return 1
}

func (g *game) samplePointer() (int, int, bool) {
	if g.pointAt == nil {
		return 0, 0, false
	}
	return g.pointAt(g.deviceScale())
}

func (g *game) syncMenuGrab() {
	if g.menuGrab == nil {
		return
	}
	g.menuGrab(g.menu)
}

func monitorSize() (int, int) {
	mon := ebiten.Monitor()
	if mon == nil {
		return 0, 0
	}
	return mon.Size()
}

func (g *game) applyWindow(screenW, screenH int) {
	if !g.haveCat || g.hidden {
		return
	}
	menuH := 0
	if g.menu {
		menuH = menuPad*2 + len(menuRows(g.ctrl.View()))*rowH
	}
	g.lay = placePet(g.menu, g.catX, g.catY, g.catW, g.catH, menuW, menuH, screenW, screenH)
	wx, wy := ebiten.WindowPosition()
	if wx != g.lay.Window.X || wy != g.lay.Window.Y {
		ebiten.SetWindowPosition(g.lay.Window.X, g.lay.Window.Y)
	}
	ww, wh := ebiten.WindowSize()
	if ww != g.lay.Window.W || wh != g.lay.Window.H {
		ebiten.SetWindowSize(g.lay.Window.W, g.lay.Window.H)
	}
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
		op.GeoM.Translate(float64(g.lay.Cat.X), float64(g.lay.Cat.Y))
		op.Filter = ebiten.FilterNearest
		screen.DrawImage(frame, op)
	}
	if !g.menu || g.lay.Menu.W <= 0 {
		return
	}
	g.pixel.Fill(color.White)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(g.lay.Menu.W), float64(g.lay.Menu.H))
	op.GeoM.Translate(float64(g.lay.Menu.X), float64(g.lay.Menu.Y))
	op.ColorScale.ScaleWithColor(color.RGBA{42, 33, 28, 235})
	screen.DrawImage(g.pixel, op)
	rows := menuRows(view)
	ink := color.RGBA{255, 246, 234, 255}
	for i, row := range rows {
		text.Draw(screen, row.label, basicfont.Face7x13, g.lay.Menu.X+12, g.lay.Menu.Y+menuPad+i*rowH+14, ink)
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
