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

// draggedOrigin is the window position after the pointer moves from
// press to now. All four values are in the same units.
func draggedOrigin(originX, originY, pressX, pressY, nowX, nowY int) (int, int) {
	return originX + nowX - pressX, originY + nowY - pressY
}

// dragAnchor is the pointer position a drag measures from.
// A button-press event wins over a later sample, so a move in the same
// instant as the press still counts. abs is false when the only position
// is the window-local cursor.
func dragAnchor(pressX, pressY int, pressed, sampled bool, sampleX, sampleY, cursorX, cursorY int) (x, y int, abs bool) {
	if pressed {
		return pressX, pressY, true
	}
	if sampled {
		return sampleX, sampleY, true
	}
	return cursorX, cursorY, false
}
