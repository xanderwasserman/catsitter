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
// window stays on the monitor area. The menu flips left or up when that side
// is the one that fits. An empty area means the monitor is unknown.
func placePet(menu bool, catX, catY, catW, catH, menuW, menuH int, area box) placement {
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
	right := area.X + area.W
	bottom := area.Y + area.H
	onLeft := false
	winX := catX
	if area.W > 0 && catX+catW+menuW > right && (catX-area.X >= menuW || catX-area.X > right-(catX+catW)) {
		onLeft = true
		winX = catX - menuW
	}
	winY := catY
	if area.H > 0 && catY+winH > bottom {
		winY = catY + catH - winH
	}
	winX = clampOrigin(winX, winW, area.X, area.W)
	winY = clampOrigin(winY, winH, area.Y, area.H)
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

// clampOrigin keeps a span of length inside [start, start+size).
// A span longer than the area keeps its start on the area.
func clampOrigin(pos, length, start, size int) int {
	if pos < start {
		pos = start
	}
	if size <= 0 {
		return pos
	}
	if length < size && pos+length > start+size {
		return start + size - length
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

// minVisible is how much of the cat, in both directions, must show on one
// monitor area for it to count as on screen when every monitor is known.
const minVisible = 24

// onScreen reports whether need units of the cat, in both directions, show on one of the areas.
func onScreen(cat box, areas []box, need int) bool {
	needW, needH := max(min(need, cat.W), 1), max(min(need, cat.H), 1)
	for _, a := range areas {
		w := min(cat.X+cat.W, a.X+a.W) - max(cat.X, a.X)
		h := min(cat.Y+cat.H, a.Y+a.H) - max(cat.Y, a.Y)
		if w >= needW && h >= needH {
			return true
		}
	}
	return false
}

// nearestArea is the area that holds the point, or else the one closest to it.
func nearestArea(areas []box, x, y int) (box, bool) {
	best, bestD := box{}, -1
	for _, a := range areas {
		if a.W <= 0 || a.H <= 0 {
			continue
		}
		dx := max(a.X-x, 0, x-(a.X+a.W-1))
		dy := max(a.Y-y, 0, y-(a.Y+a.H-1))
		d := dx*dx + dy*dy
		if bestD < 0 || d < bestD {
			best, bestD = a, d
		}
	}
	return best, bestD >= 0
}

// keepInside moves a w by h box at x, y the least distance that puts it inside area.
func keepInside(x, y, w, h int, area box) (int, int) {
	return clampOrigin(x, w, area.X, area.W), clampOrigin(y, h, area.Y, area.H)
}

// rescue brings a cat that is off every area onto the nearest one.
// moved is false when the cat already shows or no area is known.
func rescue(cat box, areas []box, need int) (x, y int, moved bool) {
	if len(areas) == 0 || onScreen(cat, areas, need) {
		return cat.X, cat.Y, false
	}
	a, ok := nearestArea(areas, cat.X+cat.W/2, cat.Y+cat.H/2)
	if !ok {
		return cat.X, cat.Y, false
	}
	x, y = keepInside(cat.X, cat.Y, cat.W, cat.H, a)
	return x, y, true
}

// keepDragOnScreen keeps a dragged cat whole on the monitor under the pointer.
// The grab point stays under the pointer except where that would push the
// cat past the monitor's edge.
func keepDragOnScreen(x, y, catW, catH, pointerX, pointerY int, areas []box) (int, int) {
	a, ok := nearestArea(areas, pointerX, pointerY)
	if !ok {
		return x, y
	}
	return keepInside(x, y, catW, catH, a)
}

// homeSpot is where a new or recalled cat sits: near the bottom right of the area.
func homeSpot(area box, catW, catH int) (int, int) {
	x, y := area.W-catW-80, area.H-catH-140
	if x < 16 {
		x = 16
	}
	if y < 16 {
		y = 16
	}
	return area.X + x, area.Y + y
}

// parkSpot is a position clear of every area, for a hidden cat.
func parkSpot(areas []box) (int, int) {
	x, y := 0, 0
	for _, a := range areas {
		x, y = min(x, a.X), min(y, a.Y)
	}
	return x - 3000, y - 3000
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

// grab is the offset that keeps one press point under the pointer.
// fromPress means x, y came from the button-press event, in window coordinates.
type grab struct {
	on        bool
	fromPress bool
	x, y      int
}

// stepGrab updates a drag for one frame.
// press is a button-press event at the window-local position px, py.
// just and cursor are the toolkit's button edge and cursor. They supply the
// offset only when this gesture has no press event.
// down is true while the button is held. A press event counts as down: the
// toolkit does not see a forwarded press until the following frame, and a
// pointer query on this frame can still report the button up.
func stepGrab(g grab, press bool, px, py int, just bool, cx, cy int, down bool) grab {
	if press {
		down = true
		g.on = true
		g.fromPress = true
		g.x, g.y = px, py
	} else if !g.fromPress && !g.on && just {
		g.on = true
		g.x, g.y = cx, cy
	}
	if !down || !g.on {
		return grab{}
	}
	return g
}
