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
// Inside clicks are forwarded to the window. open false releases the grab.
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
		c.focusWindow(win)
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
	mask := uint32(xproto.EventMaskButtonPress | xproto.EventMaskButtonRelease)
	_ = xproto.SendEventChecked(c.conn, false, win, mask, string(raw)).Check()
}

func (c *Client) focusWindow(win xproto.Window) {
	if atom, err := c.atom("_NET_ACTIVE_WINDOW"); err == nil {
		ev := xproto.ClientMessageEvent{
			Format: 32,
			Window: win,
			Type:   atom,
			Data:   xproto.ClientMessageDataUnionData32New([]uint32{1, 0, 0, 0, 0}),
		}
		mask := uint32(xproto.EventMaskSubstructureRedirect | xproto.EventMaskSubstructureNotify)
		_ = xproto.SendEventChecked(c.conn, false, c.root, mask, string(ev.Bytes())).Check()
	}
	_ = xproto.SetInputFocusChecked(c.conn, xproto.InputFocusParent, win, xproto.TimeCurrentTime).Check()
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
