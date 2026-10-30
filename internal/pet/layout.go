package pet

// box is a rectangle in window or screen pixels.
type box struct {
	X, Y, W, H int
}

func (b box) contains(x, y int) bool {
	return x >= b.X && y >= b.Y && x < b.X+b.W && y < b.Y+b.H
}

// placement is where the cat and its menu sit for one frame.
// Window is the screen rectangle. Cat and Menu are inside that window.
type placement struct {
	Window box
	Cat    box
	Menu   box
}

// placePet keeps the cat where it is and moves the menu so the whole
// window stays on the monitor. The menu flips left or up when that side
// is the one that fits.
func placePet(menu bool, catX, catY, catW, catH, menuW, menuH, screenW, screenH int) placement {
	if !menu {
		return placement{
			Window: box{catX, catY, catW, catH},
			Cat:    box{0, 0, catW, catH},
		}
	}
	winW := catW + menuW
	winH := catH
	if menuH > winH {
		winH = menuH
	}
	onLeft := false
	winX := catX
	if screenW > 0 && catX+catW+menuW > screenW && (catX >= menuW || catX > screenW-(catX+catW)) {
		onLeft = true
		winX = catX - menuW
	}
	winY := catY
	if screenH > 0 && catY+winH > screenH {
		winY = catY + catH - winH
	}
	winX = clampOrigin(winX, winW, screenW)
	winY = clampOrigin(winY, winH, screenH)
	catOffX := clampInto(catX-winX, catW, winW)
	catOffY := clampInto(catY-winY, catH, winH)
	menuX := catOffX + catW
	if onLeft {
		menuX = catOffX - menuW
	}
	menuX = clampInto(menuX, menuW, winW)
	return placement{
		Window: box{winX, winY, winW, winH},
		Cat:    box{catOffX, catOffY, catW, catH},
		Menu:   box{menuX, 0, menuW, menuH},
	}
}

func clampOrigin(pos, length, screen int) int {
	if screen <= 0 {
		if pos < 0 {
			return 0
		}
		return pos
	}
	if pos < 0 {
		pos = 0
	}
	if length < screen && pos+length > screen {
		return screen - length
	}
	return pos
}

func clampInto(pos, length, space int) int {
	if pos < 0 {
		pos = 0
	}
	if length < space && pos+length > space {
		return space - length
	}
	return pos
}

// grabbedWindow is the window origin that keeps a grab point under the pointer.
// pointer is the absolute pointer in window-position units.
// offset is the pointer's position inside the window at the press.
// The window position at the press is not an input, so a late or stale read of it cannot shift the grab.
func grabbedWindow(pointerX, pointerY, offsetX, offsetY int) (int, int) {
	return pointerX - offsetX, pointerY - offsetY
}

// grabbedWindowFromCursor is grabbedWindow when the pointer is known only as
// a cursor inside the current window. cursor is that pointer minus the window origin.
func grabbedWindowFromCursor(windowX, windowY, cursorX, cursorY, offsetX, offsetY int) (int, int) {
	return grabbedWindow(windowX+cursorX, windowY+cursorY, offsetX, offsetY)
}
