//go:build linux

package x11

import (
	"github.com/jezek/xgb/xproto"
	"github.com/xanderwasserman/catsitter/internal/platform"
)

const xkEscape = 0xff1b

// Pointer reports the pointer in window-position units.
// Root coordinates are physical pixels; deviceScale converts them.
func (c *Client) Pointer(deviceScale float64) (int, int, bool) {
	c.xmu.Lock()
	defer c.xmu.Unlock()
	qp, err := xproto.QueryPointer(c.conn, c.root).Reply()
	if err != nil {
		return 0, 0, false
	}
	return platform.DIP(int(qp.RootX), deviceScale), platform.DIP(int(qp.RootY), deviceScale), true
}

// MenuGrab takes the pointer and the keyboard while the menu is open.
// A notification window is not focused by a right-click, so Escape and a
// click on the desktop never reach it unless this process grabs them.
// The grab does not move keyboard focus. Inside clicks are forwarded to
// the window. open false releases the grab.
func (c *Client) MenuGrab(title string, open bool) bool {
	c.xmu.Lock()
	defer c.xmu.Unlock()
	if !open {
		c.releaseMenu()
		return false
	}
	win, ok := c.petWindow(title)
	if !ok {
		return false
	}
	if !c.menuArm {
		c.menuArm = true
		c.menuWin = win
		c.menuBtns = c.buttons()
		c.menuEsc = c.escapeDown()
		c.rememberEscape()
	}
	c.menuWin = win
	dismiss := c.readMenu()
	if dismiss {
		c.releaseMenu()
		return true
	}
	c.holdMenu(win)
	return false
}

func (c *Client) holdMenu(win xproto.Window) {
	if !c.menuPtr {
		reply, err := xproto.GrabPointer(
			c.conn,
			false,
			win,
			xproto.EventMaskButtonPress|xproto.EventMaskButtonRelease,
			xproto.GrabModeAsync,
			xproto.GrabModeAsync,
			0,
			0,
			xproto.TimeCurrentTime,
		).Reply()
		if err == nil && reply != nil && reply.Status == xproto.GrabStatusSuccess {
			c.menuPtr = true
		}
	}
	if !c.menuKey {
		reply, err := xproto.GrabKeyboard(
			c.conn,
			false,
			win,
			xproto.TimeCurrentTime,
			xproto.GrabModeAsync,
			xproto.GrabModeAsync,
		).Reply()
		if err == nil && reply != nil && reply.Status == xproto.GrabStatusSuccess {
			c.menuKey = true
		}
	}
}

func (c *Client) releaseMenu() {
	if c.menuPtr {
		_ = xproto.UngrabPointerChecked(c.conn, xproto.TimeCurrentTime).Check()
		c.menuPtr = false
	}
	if c.menuKey {
		_ = xproto.UngrabKeyboardChecked(c.conn, xproto.TimeCurrentTime).Check()
		c.menuKey = false
	}
	c.menuArm = false
	c.menuWin = 0
	c.menuBtns = 0
	c.menuEsc = false
}

func (c *Client) readMenu() bool {
	dismiss := false
	if c.menuPtr || c.menuKey {
		for range 32 {
			ev, err := c.conn.PollForEvent()
			if ev == nil && err == nil {
				break
			}
			if ev == nil {
				break
			}
			switch e := ev.(type) {
			case xproto.ButtonPressEvent:
				if e.Detail >= 1 && e.Detail <= 3 && !c.covers(c.menuWin, e.RootX, e.RootY) {
					dismiss = true
					continue
				}
				c.forward(c.menuWin, e.Bytes())
			case xproto.ButtonReleaseEvent:
				c.forward(c.menuWin, e.Bytes())
			case xproto.KeyPressEvent:
				if c.escKnown && e.Detail == c.escCode {
					dismiss = true
				}
			}
		}
		_, _ = xproto.GetInputFocus(c.conn).Reply()
	}
	if !c.menuPtr {
		mask := c.buttons()
		pressed := mask &^ c.menuBtns
		c.menuBtns = mask
		if pressed != 0 {
			if qp, err := xproto.QueryPointer(c.conn, c.root).Reply(); err == nil && !c.covers(c.menuWin, qp.RootX, qp.RootY) {
				dismiss = true
			}
		}
	}
	if !c.menuKey {
		down := c.escapeDown()
		if down && !c.menuEsc {
			dismiss = true
		}
		c.menuEsc = down
	}
	return dismiss
}

func (c *Client) forward(win xproto.Window, raw []byte) {
	if win == 0 || len(raw) == 0 {
		return
	}
	mask := uint32(xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease | xproto.EventMaskPointerMotion)
	_ = xproto.SendEventChecked(c.conn, false, win, mask, string(raw)).Check()
}

// Press reports button 1 in window-position units.
// pressed is a button-press event since the last call, at that event's pointer.
// down is true while the button is held. Both are false when the watch is unavailable.
func (c *Client) Press(title string, deviceScale float64) (x, y int, pressed, down bool) {
	c.xmu.Lock()
	defer c.xmu.Unlock()
	win, ok := c.petWindow(title)
	if !ok {
		return 0, 0, false, false
	}
	c.armPress(win)
	rx, ry, saw := c.drainPress()
	if !saw {
		return 0, 0, false, c.pressDown
	}
	return platform.DIP(int(rx), deviceScale), platform.DIP(int(ry), deviceScale), true, c.pressDown
}

func (c *Client) armPress(win xproto.Window) {
	if c.pressArmed && c.pressWin == win {
		return
	}
	if c.pressArmed && c.pressWin != 0 {
		_ = xproto.UngrabButtonChecked(c.conn, xproto.ButtonIndex1, c.pressWin, xproto.ModMaskAny).Check()
		c.pressArmed = false
	}
	c.pressWin = win
	mask := uint16(xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease | xproto.EventMaskPointerMotion)
	err := xproto.GrabButtonChecked(
		c.conn,
		false,
		win,
		mask,
		xproto.GrabModeAsync,
		xproto.GrabModeAsync,
		0,
		0,
		xproto.ButtonIndex1,
		xproto.ModMaskAny,
	).Check()
	if err != nil {
		return
	}
	c.pressArmed = true
}

func (c *Client) drainPress() (x, y int16, saw bool) {
	if !c.pressArmed {
		return 0, 0, false
	}
	n := 0
	for range 32 {
		ev, err := c.conn.PollForEvent()
		if ev == nil || err != nil {
			break
		}
		n++
		switch e := ev.(type) {
		case xproto.ButtonPressEvent:
			if e.Detail == 1 && !saw {
				x, y, saw = e.RootX, e.RootY, true
				c.pressDown = true
			}
			c.forward(c.pressWin, e.Bytes())
		case xproto.ButtonReleaseEvent:
			if e.Detail == 1 {
				c.pressDown = false
			}
			c.forward(c.pressWin, e.Bytes())
		case xproto.MotionNotifyEvent:
			c.forward(c.pressWin, e.Bytes())
		}
	}
	if c.pressDown && c.buttons()&xproto.ButtonMask1 == 0 {
		c.pressDown = false
	}
	if n > 0 {
		_, _ = xproto.GetInputFocus(c.conn).Reply()
	}
	return x, y, saw
}

func (c *Client) covers(win xproto.Window, rootX, rootY int16) bool {
	if win == 0 {
		return false
	}
	tr, err := xproto.TranslateCoordinates(c.conn, win, c.root, 0, 0).Reply()
	if err != nil {
		return false
	}
	geo, err := xproto.GetGeometry(c.conn, xproto.Drawable(win)).Reply()
	if err != nil {
		return false
	}
	x, y := int(rootX), int(rootY)
	left, top := int(tr.DstX), int(tr.DstY)
	return x >= left && y >= top && x < left+int(geo.Width) && y < top+int(geo.Height)
}

func (c *Client) buttons() uint16 {
	qp, err := xproto.QueryPointer(c.conn, c.root).Reply()
	if err != nil {
		return 0
	}
	return qp.Mask & (xproto.ButtonMask1 | xproto.ButtonMask2 | xproto.ButtonMask3)
}

func (c *Client) rememberEscape() {
	if c.escKnown {
		return
	}
	code, err := c.key(xkEscape)
	if err != nil {
		return
	}
	c.escCode = code
	c.escKnown = true
}

func (c *Client) escapeDown() bool {
	c.rememberEscape()
	if !c.escKnown {
		return false
	}
	reply, err := xproto.QueryKeymap(c.conn).Reply()
	if err != nil || len(reply.Keys) == 0 {
		return false
	}
	i := int(c.escCode) / 8
	if i >= len(reply.Keys) {
		return false
	}
	return reply.Keys[i]&(1<<uint(c.escCode%8)) != 0
}
