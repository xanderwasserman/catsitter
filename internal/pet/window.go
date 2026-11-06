package pet

import (
	"image/color"
	"log"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"

	"github.com/xanderwasserman/catsitter/internal/control"
	"github.com/xanderwasserman/catsitter/internal/platform"
	"github.com/xanderwasserman/catsitter/internal/tray"
	"github.com/xanderwasserman/catsitter/internal/update"
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
	// MenuGrab takes keyboard and pointer input while the menu or a card
	// opened from it is up, and reports a press outside the window or
	// Escape. Nil where the window
	// already receives those events.
	MenuGrab func(open bool) bool
	// Pointer returns the pointer in window-position units.
	// deviceScale is the monitor scale. ok is false when it is unavailable.
	Pointer func(deviceScale float64) (x, y int, ok bool)
	// Press reports where the pointer sat inside the window at the button press,
	// and whether the button is held. Nil where the window's own cursor is that position.
	Press func(deviceScale float64) (x, y int, pressed, down bool)
	// Place moves the window across the whole desktop instead of Ebitengine.
	// It also supplies the pointer, so Pointer is unused when it is set.
	Place *platform.Placer
	// NoteFocus records the app that should receive keys after the menu.
	// RestoreFocus hands them back when this process still holds the keyboard.
	// Both are nil where the menu does not take focus.
	NoteFocus    func()
	RestoreFocus func()
	// SingleThread runs Update and Draw on the main thread, so the hooks
	// and the tray start there too.
	SingleThread bool
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
		SingleThread:      desk.SingleThread,
		X11ClassName:      platform.OverlayTitle,
		X11InstanceName:   "catsitter",
	}
	pixel := ebiten.NewImage(1, 1)
	return ebiten.RunGameWithOptions(&game{
		pet:          p,
		ctrl:         ctrl,
		tray:         bar,
		catW:         catW,
		catH:         catH,
		pixel:        pixel,
		paint:        newPainter(pixel),
		goos:         runtime.GOOS,
		pump:         desk.Pump,
		overlay:      desk.Overlay,
		menuGrab:     desk.MenuGrab,
		scr:          newScreen(desk),
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
	paint        *painter
	goos         string
	pump         func()
	overlay      func() error
	menuGrab     func(open bool) bool
	scr          screen
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
	grab           grab
	scale          float64
	recall         uint64
	cards          cardState
	// drawnUpdate is the update row's action as last drawn.
	drawnUpdate control.UpdateAction
}

func (g *game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenSize()
}

func (g *game) screenSize() (int, int) {
	if g.panelShown() && g.lay.Window.W > 0 && g.lay.Window.H > 0 {
		return g.lay.Window.W, g.lay.Window.H
	}
	return g.catW, g.catH
}

func (g *game) Update() error {
	if g.noteFocus != nil {
		g.noteFocus()
	}
	g.keepOutOfSwitcher()
	if g.pump != nil {
		g.pump()
	}
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
		g.recall = view.Recall
		g.place(view)
		if g.tray != nil {
			g.tray.Start()
		}
		g.placed = true
	}
	wasGrab := g.grabbing()
	g.syncUpdates(view)
	g.syncHidden(view.Hidden && !g.cards.peek)
	if g.hidden {
		g.syncMenuGrab()
		if wasGrab {
			g.handBack()
		}
		ebiten.SetTPS(2)
		return nil
	}
	if g.panelShown() {
		ebiten.SetTPS(60)
	} else {
		ebiten.SetTPS(12)
	}
	g.followScale()
	if view.Recall != g.recall {
		g.recall = view.Recall
		g.bringBack(wasGrab)
	}
	g.pointer(view)
	g.applyWindow()
	g.keepOnScreen()
	return nil
}

func (g *game) place(view control.View) {
	g.scale = g.scr.scale()
	if view.WindowSet {
		g.catX, g.catY = g.scr.loaded(view.WindowX, view.WindowY)
		g.haveCat = true
		g.scr.move(g.catX, g.catY)
		return
	}
	g.goHome(g.homeArea())
}

// homeArea is the monitor area the window opened on.
func (g *game) homeArea() (box, bool) {
	x, y, ok := g.scr.position()
	if !ok {
		return box{}, false
	}
	return nearestArea(g.scr.areas(), x, y)
}

// goHome puts the cat at its default spot on the area and saves it.
func (g *game) goHome(area box, ok bool) {
	if !ok {
		return
	}
	x, y := homeSpot(area, g.catW, g.catH)
	g.catX, g.catY = x, y
	g.haveCat = true
	g.moveAndSave(x, y)
}

// moveAndSave converts the position for saving before the move, which can
// carry the window onto a monitor with another scale. followScale then puts
// catX and catY in the new scale's units before anything else this frame
// compares them with the window.
func (g *game) moveAndSave(x, y int) {
	sx, sy := g.scr.stored(x, y)
	g.scr.move(x, y)
	g.ctrl.SaveWindow(sx, sy)
	g.followScale()
}

// bringBack answers the tray's recall: it closes the menu and any card, ends
// any drag and puts the cat at its default spot on the monitor with the pointer.
func (g *game) bringBack(wasGrab bool) {
	g.menu = false
	g.cards.close()
	g.dragging = false
	g.dragged = false
	g.grab = grab{}
	g.syncMenuGrab()
	if wasGrab {
		g.handBack()
	}
	area, ok := g.homeArea()
	if x, y, have := g.scr.pointer(); have && g.scr.desktop() {
		area, ok = nearestArea(g.scr.areas(), x, y)
	}
	g.goHome(area, ok)
	log.Printf("cat brought back to %d,%d", g.catX, g.catY)
}

// followScale keeps the cat's position in step with the window's scale.
// g.scale is the scale catX and catY are written in.
// The units are physical pixels over the scale, so a change of scale leaves
// the old numbers describing another place. The window's own position is
// used when it is the cat's; otherwise the numbers are converted.
func (g *game) followScale() {
	s := g.scr.scale()
	old := g.scale
	if s == old {
		return
	}
	g.scale = s
	if old <= 0 || s <= 0 || !g.haveCat || g.hidden {
		return
	}
	if !g.panelShown() && !g.grab.on {
		if x, y, ok := g.scr.position(); ok {
			g.catX, g.catY = x, y
			return
		}
	}
	g.catX, g.catY = rescale(g.catX, old, s), rescale(g.catY, old, s)
}

// rescale converts a position from one scale's units to another's.
func rescale(v int, from, to float64) int {
	return platform.DIP(platform.Physical(v, from), to)
}

// keepOnScreen is the safety net: a cat that shows on no monitor is moved
// onto the nearest one and that position is saved.
func (g *game) keepOnScreen() {
	if !g.haveCat || g.hidden || g.panelShown() || g.grab.on {
		return
	}
	x, y, moved := rescue(box{g.catX, g.catY, g.catW, g.catH}, g.scr.areas(), g.scr.need())
	if !moved {
		return
	}
	log.Printf("cat was off screen at %d,%d; moved to %d,%d", g.catX, g.catY, x, y)
	g.catX, g.catY = x, y
	g.moveAndSave(x, y)
}

func (g *game) syncHidden(hidden bool) {
	if hidden == g.hidden {
		return
	}
	if hidden {
		g.savedX, g.savedY = g.scr.stored(g.catX, g.catY)
		g.menu = false
		g.cards.close()
		g.dragging = false
		g.grab = grab{}
		g.scr.move(parkSpot(g.scr.areas()))
		g.hidden = true
		return
	}
	// The window was parked, maybe on another scale. The position is read in
	// the scale it has now, and followScale catches a change from the move.
	g.scale = g.scr.scale()
	g.catX, g.catY = g.scr.loaded(g.savedX, g.savedY)
	g.haveCat = true
	g.scr.move(g.catX, g.catY)
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
	wasGrab := g.grabbing()
	inputOpen := g.inputOpen(view)
	outside := false
	if wasGrab && g.menuGrab != nil && g.menuGrab(true) {
		outside = true
	}
	open, saw, activate, consumed := nextMenu(inputOpen, menuEvent{
		Right:    inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight),
		Left:     inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft),
		Escape:   inpututil.IsKeyJustPressed(ebiten.KeyEscape),
		Outside:  outside,
		Focused:  ebiten.IsFocused(),
		SawFocus: g.menuSawFocus,
		OnMenu:   g.lay.Menu.contains(cx, cy),
	})
	g.menuSawFocus = saw
	switch {
	case activate && g.menu:
		g.menu = false
		g.activate(cy-g.lay.Menu.Y, view)
	case activate:
		g.chooseCard(cx-g.lay.Menu.X, cy-g.lay.Menu.Y, view)
	case !inputOpen && open:
		g.menu = true
	case inputOpen && !open && g.menu:
		g.menu = false
	case inputOpen && !open:
		g.cards.close()
	}
	closing := wasGrab && !g.grabbing()
	g.syncMenuGrab()
	if closing {
		g.handBack()
	}
	px, py, pressed, held := g.readPress()
	if g.inputOpen(view) || closing {
		pressed = false
		held = false
	}
	// A panel widens the window, so a press offset would no longer match the cat.
	if consumed || g.panelShown() {
		if g.dragging && g.dragged {
			g.ctrl.SaveWindow(g.scr.stored(g.catX, g.catY))
		}
		g.dragging = false
		g.dragged = false
		g.grab = grab{}
		return
	}
	if !g.haveCat {
		return
	}
	just := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	down := held || pressed || ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	was := g.grab.on
	gx, gy := cx, cy
	if just && !was && g.scr.desktop() {
		gx, gy = g.localPointer(cx, cy)
	}
	g.grab = stepGrab(g.grab, pressed, px, py, just, gx, gy, down)
	if g.grab.on && !was {
		g.dragging = true
		g.dragged = false
		g.dragAbs = g.scr.hasPointer()
	}
	if g.grab.on {
		g.dragging = true
		g.moveDrag(cx, cy)
		return
	}
	if g.dragging && g.dragged {
		g.ctrl.SaveWindow(g.scr.stored(g.catX, g.catY))
	} else if g.dragging && g.badgeShown(view) {
		g.cards.show(cardOffer, true, false)
		g.menuSawFocus = ebiten.IsFocused()
	}
	g.dragging = false
	g.dragged = false
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
	return g.buttonAt(deviceScale())
}

// localPointer is the pointer inside the window, read in the same units as
// the window position. Ebitengine's cursor uses its own idea of the monitor
// scale, which lags a move onto another monitor. cx, cy is that cursor.
func (g *game) localPointer(cx, cy int) (int, int) {
	px, py, ok := g.scr.pointer()
	wx, wy, have := g.scr.position()
	if !ok || !have {
		return cx, cy
	}
	return px - wx, py - wy
}

func (g *game) moveDrag(cx, cy int) {
	var nx, ny int
	if g.dragAbs {
		x, y, ok := g.scr.pointer()
		if !ok {
			return
		}
		nx, ny = grabbedWindow(x, y, g.grab.x, g.grab.y)
		if g.scr.desktop() {
			nx, ny = keepDragOnScreen(nx, ny, g.catW, g.catH, x, y, g.scr.areas())
		}
	} else {
		wx, wy := ebiten.WindowPosition()
		nx, ny = grabbedWindowFromCursor(wx, wy, cx, cy, g.grab.x, g.grab.y)
	}
	if nx != g.catX || ny != g.catY {
		g.dragged = true
	}
	g.catX, g.catY = nx, ny
}

func (g *game) syncMenuGrab() {
	if g.menuGrab == nil {
		return
	}
	g.menuGrab(g.grabbing())
}

func (g *game) applyWindow() {
	if !g.haveCat || g.hidden {
		return
	}
	panelW, panelH := g.panelSize(g.ctrl.View())
	area, _ := nearestArea(g.scr.areas(), g.catX+g.catW/2, g.catY+g.catH/2)
	g.lay = placePet(g.panelShown(), g.catX, g.catY, g.catW, g.catH, panelW, panelH, area)
	wx, wy, ok := g.scr.position()
	if !ok || wx != g.lay.Window.X || wy != g.lay.Window.Y {
		g.scr.move(g.lay.Window.X, g.lay.Window.Y)
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
	case "access":
		g.ctrl.RequestAccess()
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
	case "update":
		g.runUpdateItem(view)
	case "about":
		g.cards.show(cardAbout, true, false)
		g.menuSawFocus = ebiten.IsFocused()
	case "quit":
		g.ctrl.Quit()
	}
}

func (g *game) Draw(screen *ebiten.Image) {
	view := g.ctrl.View()
	if g.hidden {
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
	if !g.menu && g.cards.kind != cardNone && g.lay.Menu.W > 0 {
		cx, cy := ebiten.CursorPosition()
		tailLeft := g.lay.Menu.X > g.lay.Cat.X
		tailY := g.lay.Cat.Y + g.catH/3 - g.lay.Menu.Y
		g.paint.card(screen, g.currentCard(view), g.lay.Menu, tailLeft, tailY, cx, cy)
		return
	}
	if !g.menu && g.badgeShown(view) {
		bw := len(badge[0]) * badgeScale
		g.paint.badge(screen, g.lay.Cat.X+g.catW-bw-4, g.lay.Cat.Y+6)
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
	_, g.drawnUpdate = control.UpdateItem(view.Update)
	for i, row := range rows {
		ink := inkText
		if row.hot {
			ink = inkFurHot
		}
		text.Draw(screen, row.label, basicfont.Face7x13, g.lay.Menu.X+12, g.lay.Menu.Y+menuPad+i*rowH+14, ink)
	}
}

func activity(view control.View, tick int64) Activity {
	if s := view.Update.State; s == update.Downloading || s == update.Installing {
		return ActivityPaws
	}
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
	hot   bool
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
	}
	if label, _ := control.UpdateItem(view.Update); label != "" {
		rows = append(rows, menuRow{id: "update", label: label, hot: view.Update.State == update.Available})
	}
	rows = append(rows,
		menuRow{id: "about", label: "About Catsitter"},
		menuRow{id: "quit", label: "Quit"},
	)
	if view.Trusted {
		return rows
	}
	access := []menuRow{
		{id: "paused", label: "Nudges paused (no access)"},
		{id: "access", label: "Allow Accessibility..."},
	}
	return append(access, rows...)
}
