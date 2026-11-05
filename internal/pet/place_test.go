package pet

import (
	"math"
	"testing"

	"github.com/xanderwasserman/catsitter/internal/platform"
)

// A docked work laptop: a 1920x1080 external monitor at 100% is primary, and
// the 2560x1600 laptop panel at 150% sits to its right, 200 pixels lower.
// Units are physical pixels over the pet window's scale, as on Windows.
func dockedAreas(scale float64) []box {
	ext := platform.RectDIP(0, 0, 1920, 1080, scale)
	lap := platform.RectDIP(1920, 200, 1920+2560, 200+1600, scale)
	return []box{{ext.X, ext.Y, ext.W, ext.H}, {lap.X, lap.Y, lap.W, lap.H}}
}

// ebitenPhysical is where Ebitengine's SetWindowPosition puts the window:
// x, y count from the top left of the window's monitor, in its units.
func ebitenPhysical(x, y, monX, monY int, scale float64) (int, int) {
	return monX + platform.Physical(x, scale), monY + platform.Physical(y, scale)
}

func TestDesktopDragKeepsTheGrabUnderThePointer(t *testing.T) {
	const catW, catH, offX, offY = 96, 96, 40, 30
	for _, scale := range []float64{1, 1.25, 1.5, 1.75} {
		// The cursor on the laptop panel, which does not start at 0,0.
		for _, cur := range [][2]int{{2400, 900}, {3001, 1203}, {4100, 1500}} {
			px, py := platform.DIP(cur[0], scale), platform.DIP(cur[1], scale)
			x, y := grabbedWindow(px, py, offX, offY)
			x, y = keepDragOnScreen(x, y, catW, catH, px, py, dockedAreas(scale))
			wx, wy := platform.Physical(x, scale), platform.Physical(y, scale)
			gx := float64(wx) + offX*scale
			gy := float64(wy) + offY*scale
			if math.Abs(gx-float64(cur[0])) > scale || math.Abs(gy-float64(cur[1])) > scale {
				t.Fatalf("scale %v cursor %v: grab point at %.0f,%.0f", scale, cur, gx, gy)
			}
		}
	}
}

// The 0.1.2 drag fed desktop units to the monitor-relative SetWindowPosition.
// On any monitor but the primary that moved the cat by the monitor's origin,
// off every screen.
func TestMonitorRelativeDragLostTheCat(t *testing.T) {
	const scale = 1.5
	const curX, curY = 3000, 900
	px, py := platform.DIP(curX, scale), platform.DIP(curY, scale)
	x, y := grabbedWindow(px, py, 40, 30)
	wx, wy := ebitenPhysical(x, y, 1920, 200, scale)
	got := box{platform.DIP(wx, scale), platform.DIP(wy, scale), 96, 96}
	if onScreen(got, dockedAreas(scale), minVisible) {
		t.Fatalf("expected the old maths to lose the cat, got %+v", got)
	}
	if _, _, moved := rescue(got, dockedAreas(scale), minVisible); !moved {
		t.Fatal("the safety net left a lost cat alone")
	}
}

func TestDragStopsAtTheMonitorEdge(t *testing.T) {
	areas := []box{{0, 0, 1920, 1040}}
	// The pointer is at the right edge; the grab point is the cat's left side.
	x, y := keepDragOnScreen(1915, 500, 96, 96, 1919, 504, areas)
	if x != 1920-96 || y != 500 {
		t.Fatalf("cat at %d,%d", x, y)
	}
	// Below the work area, above the taskbar.
	_, y = keepDragOnScreen(500, 1000, 96, 96, 540, 1039, areas)
	if y != 1040-96 {
		t.Fatalf("cat y %d", y)
	}
}

func TestDragFollowsThePointerOntoANegativeMonitor(t *testing.T) {
	// A monitor to the left of the primary has negative coordinates.
	areas := []box{{-1920, 0, 1920, 1080}, {0, 0, 1707, 1067}}
	x, y := keepDragOnScreen(-700, 400, 96, 96, -660, 430, areas)
	if x != -700 || y != 400 {
		t.Fatalf("cat at %d,%d", x, y)
	}
	// Across the seam the cat stays whole on the pointer's monitor.
	x, _ = keepDragOnScreen(-50, 400, 96, 96, -10, 430, areas)
	if x != -96 {
		t.Fatalf("cat x %d", x)
	}
}

func TestRescueLeavesAVisibleCat(t *testing.T) {
	areas := dockedAreas(1)
	for _, c := range []box{
		{100, 100, 96, 96},
		{-60, 500, 96, 96},   // two thirds past the left edge
		{1880, 50, 96, 96},   // 40 units on the external, the rest over the gap
		{3000, 1750, 96, 96}, // over the laptop's bottom edge
	} {
		if x, y, moved := rescue(c, areas, minVisible); moved {
			t.Fatalf("%+v moved to %d,%d", c, x, y)
		}
	}
}

func TestRescueBringsBackALostCat(t *testing.T) {
	areas := dockedAreas(1)
	for _, c := range []box{
		{-3000, -3000, 96, 96},
		{5000, 900, 96, 96},
		{1900, -200, 96, 96},
		{1950, 100, 96, 96}, // the gap above the lower laptop panel
		{-90, 500, 96, 96},  // only 6 units show
	} {
		x, y, moved := rescue(c, areas, minVisible)
		if !moved {
			t.Fatalf("%+v stayed lost", c)
		}
		got := box{x, y, c.W, c.H}
		inside := false
		for _, a := range areas {
			if got.X >= a.X && got.Y >= a.Y && got.X+got.W <= a.X+a.W && got.Y+got.H <= a.Y+a.H {
				inside = true
			}
		}
		if !inside {
			t.Fatalf("%+v rescued to %+v, not wholly on a monitor", c, got)
		}
	}
}

func TestRescueNeedsAMonitor(t *testing.T) {
	if _, _, moved := rescue(box{-3000, -3000, 96, 96}, nil, minVisible); moved {
		t.Fatal("moved without a known monitor")
	}
}

func TestNearestAreaPrefersTheOneHoldingThePoint(t *testing.T) {
	areas := dockedAreas(1)
	if a, _ := nearestArea(areas, 2000, 900); a != areas[1] {
		t.Fatalf("got %+v", a)
	}
	if a, _ := nearestArea(areas, 1950, 100); a != areas[0] {
		t.Fatalf("gap point: got %+v", a)
	}
	if _, ok := nearestArea(nil, 0, 0); ok {
		t.Fatal("found an area in an empty list")
	}
}

func TestHomeAndParkSpots(t *testing.T) {
	// One monitor at the origin keeps the 0.1.2 spots.
	if x, y := homeSpot(box{0, 0, 1920, 1080}, 96, 96); x != 1744 || y != 844 {
		t.Fatalf("home %d,%d", x, y)
	}
	if x, y := parkSpot([]box{{0, 0, 1920, 1080}}); x != -3000 || y != -3000 {
		t.Fatalf("park %d,%d", x, y)
	}
	// A monitor at negative coordinates moves both with it.
	if x, y := homeSpot(box{-1920, -300, 1920, 1080}, 96, 96); x != -176 || y != 544 {
		t.Fatalf("home %d,%d", x, y)
	}
	areas := []box{{-1920, -300, 1920, 1080}, {0, 0, 1920, 1080}}
	x, y := parkSpot(areas)
	if onScreen(box{x, y, 96, 96}, areas, minVisible) {
		t.Fatalf("parked cat at %d,%d shows", x, y)
	}
}

func TestMenuStaysOnAnOffsetMonitor(t *testing.T) {
	area := box{-1920, 200, 1920, 1040}
	const catX, catY = -100, 1140
	menuH := menuPad*2 + 9*rowH
	lay := placePet(true, catX, catY, 96, 96, menuW, menuH, area)
	w := lay.Window
	if w.X < area.X || w.Y < area.Y || w.X+w.W > area.X+area.W || w.Y+w.H > area.Y+area.H {
		t.Fatalf("window %+v leaves %+v", w, area)
	}
	if x, y := w.X+lay.Cat.X, w.Y+lay.Cat.Y; x != catX || y != catY {
		t.Fatalf("cat moved to %d,%d", x, y)
	}
	if lay.Menu.X >= lay.Cat.X {
		t.Fatalf("menu should open left: %+v", lay)
	}
}

func TestOnlyACatWithNothingShowingIsLostToOneMonitor(t *testing.T) {
	// Without the full monitor list, a cat across a seam is left alone.
	areas := []box{{0, 0, 1920, 1080}}
	if _, _, moved := rescue(box{1910, 500, 96, 96}, areas, 1); moved {
		t.Fatal("moved a cat that still shows")
	}
	if _, _, moved := rescue(box{1920, 500, 96, 96}, areas, 1); !moved {
		t.Fatal("left a cat with nothing showing")
	}
}
