//go:build linux

// Package x11 reads idle time and posts synthetic input through XTEST.
// Wayland sessions without XWayland have no DISPLAY connection; callers
// should keep the sleep inhibitor and skip nudges when Open fails.
package x11

import (
	"fmt"
	"sync"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/screensaver"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgb/xtest"
)

const (
	xkShiftL = 0xffe1
	xkAltL   = 0xffe9
	xkSuperL = 0xffeb
	xkTab    = 0xff09
)

// Client is one X connection shared by idle detection and input.
type Client struct {
	conn *xgb.Conn
	root xproto.Window
	ss   bool
	keys map[uint32]xproto.Keycode

	mu       sync.Mutex
	lastX    int16
	lastY    int16
	havePos  bool
	lastMove time.Time
}

// Open connects to the server in DISPLAY.
func Open() (*Client, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}
	if err := xtest.Init(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("XTEST: %w", err)
	}
	setup := xproto.Setup(conn)
	c := &Client{
		conn:     conn,
		root:     setup.DefaultScreen(conn).Root,
		keys:     map[uint32]xproto.Keycode{},
		lastMove: time.Now(),
	}
	if err := screensaver.Init(conn); err == nil {
		c.ss = true
	}
	if err := c.cacheKeys(setup); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

// Close closes the X connection.
func (c *Client) Close() { c.conn.Close() }

func (c *Client) cacheKeys(setup *xproto.SetupInfo) error {
	min, max := setup.MinKeycode, setup.MaxKeycode
	reply, err := xproto.GetKeyboardMapping(c.conn, min, byte(max-min+1)).Reply()
	if err != nil {
		return err
	}
	per := int(reply.KeysymsPerKeycode)
	for kc := int(min); kc <= int(max); kc++ {
		base := (kc - int(min)) * per
		if base >= len(reply.Keysyms) {
			break
		}
		sym := uint32(reply.Keysyms[base])
		if sym != 0 {
			if _, ok := c.keys[sym]; !ok {
				c.keys[sym] = xproto.Keycode(kc)
			}
		}
	}
	return nil
}

// IdleTime prefers the X screensaver idle counter and falls back to how
// long the pointer has stayed still.
func (c *Client) IdleTime() (time.Duration, error) {
	if c.ss {
		reply, err := screensaver.QueryInfo(c.conn, xproto.Drawable(c.root)).Reply()
		if err == nil {
			return time.Duration(reply.MsSinceUserInput) * time.Millisecond, nil
		}
	}
	return c.pointerIdle()
}

func (c *Client) pointerIdle() (time.Duration, error) {
	qp, err := xproto.QueryPointer(c.conn, c.root).Reply()
	if err != nil {
		return 0, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.havePos || qp.RootX != c.lastX || qp.RootY != c.lastY {
		c.lastX, c.lastY = qp.RootX, qp.RootY
		c.havePos = true
		c.lastMove = time.Now()
	}
	return time.Since(c.lastMove), nil
}

// Jiggle moves the pointer by pixels and back.
func (c *Client) Jiggle(pixels int) error {
	if pixels < 1 {
		pixels = 1
	}
	if err := c.relMove(int16(pixels), 0); err != nil {
		return err
	}
	time.Sleep(30 * time.Millisecond)
	return c.relMove(int16(-pixels), 0)
}

// TapShift presses and releases left Shift.
func (c *Client) TapShift() error {
	code, err := c.key(xkShiftL)
	if err != nil {
		return err
	}
	if err := c.keyEvent(xproto.KeyPress, code); err != nil {
		return err
	}
	time.Sleep(30 * time.Millisecond)
	return c.keyEvent(xproto.KeyRelease, code)
}

// SwitchWindows holds Alt or Super and taps Tab count times.
func (c *Client) SwitchWindows(modifier string, count int) error {
	if count < 1 {
		count = 1
	}
	sym := uint32(xkAltL)
	if modifier == "cmd" {
		sym = xkSuperL
	}
	mod, err := c.key(sym)
	if err != nil {
		return err
	}
	tab, err := c.key(xkTab)
	if err != nil {
		return err
	}
	if err := c.keyEvent(xproto.KeyPress, mod); err != nil {
		return err
	}
	time.Sleep(20 * time.Millisecond)
	for i := 0; i < count; i++ {
		if err := c.keyEvent(xproto.KeyPress, tab); err != nil {
			return err
		}
		time.Sleep(30 * time.Millisecond)
		if err := c.keyEvent(xproto.KeyRelease, tab); err != nil {
			return err
		}
		time.Sleep(40 * time.Millisecond)
	}
	return c.keyEvent(xproto.KeyRelease, mod)
}

func (c *Client) key(sym uint32) (xproto.Keycode, error) {
	code, ok := c.keys[sym]
	if !ok || code == 0 {
		return 0, fmt.Errorf("keysym %#x is not on this keyboard", sym)
	}
	return code, nil
}

func (c *Client) relMove(dx, dy int16) error {
	// root 0 means a relative motion for XTEST MotionNotify.
	xtest.FakeInput(c.conn, xproto.MotionNotify, 0, xproto.TimeCurrentTime, 0, dx, dy, 0)
	_, err := xproto.GetInputFocus(c.conn).Reply()
	return err
}

func (c *Client) keyEvent(kind byte, code xproto.Keycode) error {
	xtest.FakeInput(c.conn, kind, byte(code), xproto.TimeCurrentTime, c.root, 0, 0, 0)
	_, err := xproto.GetInputFocus(c.conn).Reply()
	return err
}
