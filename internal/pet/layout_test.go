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

func TestDragTracksThePointer(t *testing.T) {
	x, y := draggedOrigin(1744, 964, 1000, 500, 200, 0)
	if x != 944 || y != 464 {
		t.Fatalf("%d,%d", x, y)
	}
}
