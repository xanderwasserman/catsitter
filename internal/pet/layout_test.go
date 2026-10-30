package pet

import "testing"

func TestMenuStaysOnScreenAtBottomRight(t *testing.T) {
	const (
		cat     = 96
		rows    = 9
		screenW = 1920
		screenH = 1080
		catX    = 1744
		catY    = 964
	)
	menuH := menuPad*2 + rows*rowH
	lay := placePet(true, catX, catY, cat, cat, menuW, menuH, screenW, screenH)
	if lay.Window.X < 0 || lay.Window.Y < 0 || lay.Window.X+lay.Window.W > screenW || lay.Window.Y+lay.Window.H > screenH {
		t.Fatalf("window %+v is off the screen", lay.Window)
	}
	if x, y := lay.Window.X+lay.Cat.X, lay.Window.Y+lay.Cat.Y; x != catX || y != catY {
		t.Fatalf("cat moved to %d,%d", x, y)
	}
	if lay.Menu.X+lay.Menu.W > lay.Window.W || lay.Menu.Y+lay.Menu.H > lay.Window.H {
		t.Fatalf("menu %+v outside window %+v", lay.Menu, lay.Window)
	}
	if lay.Menu.X >= lay.Cat.X {
		t.Fatalf("menu should sit left of the cat: menu %+v cat %+v", lay.Menu, lay.Cat)
	}
	if lay.Cat.Y <= lay.Menu.Y {
		t.Fatalf("menu should sit above the cat's bottom: menu %+v cat %+v", lay.Menu, lay.Cat)
	}
}

func TestMenuOpensRightWhenThereIsRoom(t *testing.T) {
	lay := placePet(true, 100, 80, 96, 96, menuW, 40, 1920, 1080)
	if lay.Window.X != 100 || lay.Window.Y != 80 {
		t.Fatalf("window %+v", lay.Window)
	}
	if lay.Cat.X != 0 || lay.Menu.X != 96 {
		t.Fatalf("cat %+v menu %+v", lay.Cat, lay.Menu)
	}
}

func TestGrabbedWindowKeepsTheGrabPoint(t *testing.T) {
	// The pointer moved 22,-18 in the instant after the press, before the drag ran.
	const (
		winX    = 1744
		winY    = 964
		offsetX = 40
		offsetY = 30
		moveX   = 22
		moveY   = -18
	)
	pointerX := winX + offsetX + moveX
	pointerY := winY + offsetY + moveY
	gotX, gotY := grabbedWindow(pointerX, pointerY, offsetX, offsetY)
	if gotX != winX+moveX || gotY != winY+moveY {
		t.Fatalf("window %d,%d", gotX, gotY)
	}
	if gotX+offsetX != pointerX || gotY+offsetY != pointerY {
		t.Fatalf("grab point %d,%d left pointer %d,%d", gotX+offsetX, gotY+offsetY, pointerX, pointerY)
	}
}

func TestPressAnchorSurvivesTheFollowingCursor(t *testing.T) {
	var g grab
	// The pointer has already stepped 13,7 when the press frame runs.
	// The toolkit button edge arrives on the next frame, another step later.
	g = stepGrab(g, true, 48, 48, false, 61, 55, false)
	g = stepGrab(g, false, 0, 0, true, 74, 62, true)
	g = stepGrab(g, false, 0, 0, false, 87, 69, true)
	if !g.on || !g.fromPress || g.x != 48 || g.y != 48 {
		t.Fatalf("anchor %+v", g)
	}
	const winX, winY = 400, 300
	pointerX := winX + 48 + 13*3
	pointerY := winY + 48 + 7*3
	gotX, gotY := grabbedWindow(pointerX, pointerY, g.x, g.y)
	if gotX+g.x != pointerX || gotY+g.y != pointerY {
		t.Fatalf("grab point %d,%d pointer %d,%d", gotX+g.x, gotY+g.y, pointerX, pointerY)
	}
}

func TestLatePressReplacesTheCursorAnchor(t *testing.T) {
	var g grab
	g = stepGrab(g, false, 0, 0, true, 61, 55, true)
	g = stepGrab(g, true, 48, 48, false, 74, 62, true)
	if !g.fromPress || g.x != 48 || g.y != 48 {
		t.Fatalf("anchor %+v", g)
	}
}

func TestCursorAnchorWhenThereIsNoPressEvent(t *testing.T) {
	var g grab
	g = stepGrab(g, false, 0, 0, true, 40, 30, true)
	g = stepGrab(g, false, 0, 0, false, 61, 55, true)
	if !g.on || g.fromPress || g.x != 40 || g.y != 30 {
		t.Fatalf("anchor %+v", g)
	}
	g = stepGrab(g, false, 0, 0, false, 61, 55, false)
	if g.on {
		t.Fatal("grab stayed down after release")
	}
}

func TestCursorDragMatchesTheAbsoluteGrab(t *testing.T) {
	const (
		winX, winY = 1744, 964
		curX, curY = 55, 28
		offX, offY = 40, 30
	)
	ax, ay := grabbedWindow(winX+curX, winY+curY, offX, offY)
	bx, by := grabbedWindowFromCursor(winX, winY, curX, curY, offX, offY)
	if ax != bx || ay != by {
		t.Fatalf("absolute %d,%d cursor %d,%d", ax, ay, bx, by)
	}
}
