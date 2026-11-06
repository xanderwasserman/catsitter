package pet

import (
	"testing"
	"time"

	"github.com/xanderwasserman/catsitter/internal/control"
	"github.com/xanderwasserman/catsitter/internal/platform"
)

// fakeMonitor is a monitor in physical pixels with its DPI scale.
type fakeMonitor struct {
	left, top, right, bottom int
	scale                    float64
}

// fakeDesk is a Windows desktop as the Win32 placer sees it. The window's
// scale follows the monitor under its centre, as GetDpiForWindow does.
type fakeDesk struct {
	mons       []fakeMonitor
	winX, winY int // physical
	catW       int // units
	cursorX    int // physical
	cursorY    int
}

func (d *fakeDesk) scale() float64 {
	half := float64(d.catW) / 2
	for _, m := range d.mons {
		cx := d.winX + int(half*m.scale)
		cy := d.winY + int(half*m.scale)
		if cx >= m.left && cx < m.right && cy >= m.top && cy < m.bottom {
			return m.scale
		}
	}
	return d.mons[0].scale
}

func (d *fakeDesk) placer() *platform.Placer {
	return &platform.Placer{
		Scale: func(float64) float64 { return d.scale() },
		Position: func(float64) (int, int, bool) {
			s := d.scale()
			return platform.DIP(d.winX, s), platform.DIP(d.winY, s), true
		},
		Move: func(x, y int, _ float64) {
			s := d.scale()
			d.winX, d.winY = platform.Physical(x, s), platform.Physical(y, s)
		},
		Pointer: func(float64) (int, int, bool) {
			s := d.scale()
			return platform.DIP(d.cursorX, s), platform.DIP(d.cursorY, s), true
		},
		Areas: func(float64) []platform.Rect {
			s := d.scale()
			var out []platform.Rect
			for _, m := range d.mons {
				out = append(out, platform.RectDIP(m.left, m.top, m.right, m.bottom, s))
			}
			return out
		},
	}
}

type fakeCtrl struct {
	view  control.View
	saves int
}

func (c *fakeCtrl) View() control.View        { return c.view }
func (c *fakeCtrl) TogglePause()              {}
func (c *fakeCtrl) SetInhibit(bool)           {}
func (c *fakeCtrl) SetMouse(bool)             {}
func (c *fakeCtrl) SetShift(bool)             {}
func (c *fakeCtrl) SetSwitch(bool)            {}
func (c *fakeCtrl) SetInterval(time.Duration) {}
func (c *fakeCtrl) SetDuration(time.Duration) {}
func (c *fakeCtrl) ToggleHidden()             {}
func (c *fakeCtrl) BringBack()                {}
func (c *fakeCtrl) RequestAccess()            {}
func (c *fakeCtrl) CheckForUpdates()          {}
func (c *fakeCtrl) InstallUpdate()            {}
func (c *fakeCtrl) SetAutoUpdate(bool)        {}
func (c *fakeCtrl) ShowAbout()                {}
func (c *fakeCtrl) Quit()                     {}
func (c *fakeCtrl) SaveWindow(x, y int) {
	c.view.WindowX, c.view.WindowY, c.view.WindowSet = x, y, true
	c.saves++
}

// A 150% laptop panel is primary; a 100% monitor sits to its right.
func mixedDesk() *fakeDesk {
	return &fakeDesk{
		mons: []fakeMonitor{
			{0, 0, 2560, 1600, 1.5},
			{2560, 0, 4480, 1080, 1},
		},
		catW: 96,
	}
}

func deskGame(d *fakeDesk, ctrl *fakeCtrl) *game {
	return &game{
		ctrl: ctrl,
		catW: 96,
		catH: 96,
		scr:  placerScreen{p: d.placer(), fallback: func() float64 { return 1 }},
	}
}

// One frame of the position bookkeeping that Update runs.
func frame(g *game) {
	g.followScale()
	frameRest(g)
}

// frameRest is what Update runs after followScale and the recall.
func frameRest(g *game) {
	if g.haveCat && !g.hidden {
		if x, y, ok := g.scr.position(); !ok || x != g.catX || y != g.catY {
			g.scr.move(g.catX, g.catY)
		}
	}
	g.keepOnScreen()
}

// near is within the one physical pixel a trip through 150% units can round.
func near(d *fakeDesk, x, y int) bool {
	return d.winX >= x-1 && d.winX <= x+1 && d.winY >= y-1 && d.winY <= y+1
}

func TestSavedSpotSurvivesARestartOnAnotherScale(t *testing.T) {
	d := mixedDesk()
	ctrl := &fakeCtrl{}
	// The cat sits on the 100% monitor at physical 3500,400 and is saved there.
	d.winX, d.winY = 3500, 400
	g := deskGame(d, ctrl)
	g.haveCat = true
	g.scale = g.scr.scale()
	g.catX, g.catY, _ = g.scr.position()
	g.moveAndSave(g.catX, g.catY)
	saved := ctrl.view

	// Next launch the window opens on the 150% laptop, where the cursor is.
	d.winX, d.winY = 1000, 600
	ctrl = &fakeCtrl{view: saved}
	g = deskGame(d, ctrl)
	g.place(saved)
	for range 3 {
		frame(g)
	}
	if !near(d, 3500, 400) {
		t.Fatalf("window at %d,%d, want 3500,400", d.winX, d.winY)
	}
	if ctrl.saves != 0 {
		t.Fatalf("the safety net moved a cat that was on screen (%d saves)", ctrl.saves)
	}
}

func TestHideAndShowReturnToTheSameSpot(t *testing.T) {
	d := mixedDesk()
	ctrl := &fakeCtrl{}
	d.winX, d.winY = 3500, 400
	g := deskGame(d, ctrl)
	g.haveCat = true
	g.scale = g.scr.scale()
	g.catX, g.catY, _ = g.scr.position()

	g.syncHidden(true)
	if d.scale() != 1.5 {
		t.Fatalf("parked window should pick up the laptop's scale, got %v", d.scale())
	}
	g.syncHidden(false)
	for range 3 {
		frame(g)
	}
	if !near(d, 3500, 400) {
		t.Fatalf("window at %d,%d, want 3500,400", d.winX, d.winY)
	}
}

func TestScaleChangeOnTheReleaseFrameKeepsTheCat(t *testing.T) {
	d := mixedDesk()
	ctrl := &fakeCtrl{}
	// The last drag frame moved the window onto the 100% monitor; the grab
	// is still on when the next frame sees the new scale.
	d.winX, d.winY = 1000, 600
	g := deskGame(d, ctrl)
	g.haveCat = true
	g.scale = g.scr.scale()
	g.catX, g.catY = platform.DIP(3000, 1.5), platform.DIP(500, 1.5)
	g.scr.move(g.catX, g.catY)
	g.grab = grab{on: true}
	frame(g)
	g.grab = grab{}
	frame(g)
	if !near(d, 3000, 500) {
		t.Fatalf("window jumped to %d,%d", d.winX, d.winY)
	}
}

func TestSafetyNetBringsBackAWindowOffEveryMonitor(t *testing.T) {
	d := mixedDesk()
	ctrl := &fakeCtrl{}
	d.winX, d.winY = 6000, 300
	g := deskGame(d, ctrl)
	g.haveCat = true
	g.scale = g.scr.scale()
	g.catX, g.catY, _ = g.scr.position()
	frame(g)
	frame(g)
	if d.winX < 2560 || d.winX+96 > 4480 {
		t.Fatalf("window still off screen at %d,%d", d.winX, d.winY)
	}
	if ctrl.saves == 0 {
		t.Fatal("the rescued spot was not saved")
	}
}

func TestBringBackOntoAMonitorWithAnotherScale(t *testing.T) {
	d := mixedDesk()
	ctrl := &fakeCtrl{}
	// The cat is on the 100% monitor; the pointer is at the tray on the 150% laptop.
	d.winX, d.winY = 3500, 400
	d.cursorX, d.cursorY = 2000, 1500
	g := deskGame(d, ctrl)
	g.haveCat = true
	g.scale = g.scr.scale()
	g.catX, g.catY, _ = g.scr.position()
	frame(g)
	// The order Update uses: followScale, the recall, then the rest.
	g.followScale()
	g.bringBack(false)
	frameRest(g)
	frame(g)
	// The spot is worked out in the units of the cat's monitor at the
	// recall, 100%, where the laptop spans 0,0 to 2560,1600.
	wantX, wantY := homeSpot(box{0, 0, 2560, 1600}, 96, 96)
	if !onScreen(box{d.winX, d.winY, 144, 144}, []box{{0, 0, 2560, 1600}}, 144) {
		t.Fatalf("window at %d,%d is not wholly on the laptop", d.winX, d.winY)
	}
	if !near(d, wantX, wantY) {
		t.Fatalf("window at %d,%d, want %d,%d", d.winX, d.winY, wantX, wantY)
	}
	if !near(&fakeDesk{winX: ctrl.view.WindowX, winY: ctrl.view.WindowY}, wantX, wantY) {
		t.Fatalf("saved %d,%d, want %d,%d", ctrl.view.WindowX, ctrl.view.WindowY, wantX, wantY)
	}
}
