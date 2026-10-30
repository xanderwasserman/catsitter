//go:build linux

// Package x11 reads idle time and posts synthetic input through XTEST.
// Wayland sessions without XWayland have no DISPLAY connection; callers
// should keep the sleep inhibitor and skip nudges when Open fails.
package x11

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/screensaver"
	"github.com/jezek/xgb/xproto"
	"github.com/jezek/xgb/xtest"
	"github.com/xanderwasserman/catsitter/internal/platform"
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
	xmu      sync.Mutex
	atoms    map[string]xproto.Atom
	petWin   xproto.Window
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
	c.xmu.Lock()
	defer c.xmu.Unlock()
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
	return platform.HoldAndTap(
		func() error { return c.keyEvent(xproto.KeyPress, mod) },
		func() error { return c.keyEvent(xproto.KeyRelease, mod) },
		func() error { return c.keyEvent(xproto.KeyPress, tab) },
		func() error { return c.keyEvent(xproto.KeyRelease, tab) },
		count,
		func() { time.Sleep(20 * time.Millisecond) },
		func() { time.Sleep(30 * time.Millisecond) },
		func() { time.Sleep(40 * time.Millisecond) },
	)
}

func (c *Client) key(sym uint32) (xproto.Keycode, error) {
	code, ok := c.keys[sym]
	if !ok || code == 0 {
		return 0, fmt.Errorf("keysym %#x is not on this keyboard", sym)
	}
	return code, nil
}

func (c *Client) relMove(dx, dy int16) error {
	c.xmu.Lock()
	defer c.xmu.Unlock()
	eventType, detail, root, x, y := relativeMotion(dx, dy)
	xtest.FakeInput(c.conn, eventType, detail, xproto.TimeCurrentTime, root, x, y, 0)
	_, err := xproto.GetInputFocus(c.conn).Reply()
	return err
}

func (c *Client) keyEvent(kind byte, code xproto.Keycode) error {
	c.xmu.Lock()
	defer c.xmu.Unlock()
	xtest.FakeInput(c.conn, kind, byte(code), xproto.TimeCurrentTime, c.root, 0, 0, 0)
	_, err := xproto.GetInputFocus(c.conn).Reply()
	return err
}

// ExcludeFromSwitcher marks the pet window so the taskbar, pager, and
// Alt+Tab skip it. The window type is a notification, which window
// managers leave out of the switcher.
func (c *Client) ExcludeFromSwitcher(title string) error {
	c.xmu.Lock()
	defer c.xmu.Unlock()
	win, ok := c.petWindow(title)
	if !ok {
		return fmt.Errorf("window %q was not found", title)
	}
	typ, err := c.atom("_NET_WM_WINDOW_TYPE")
	if err != nil {
		return err
	}
	note, err := c.atom("_NET_WM_WINDOW_TYPE_NOTIFICATION")
	if err != nil {
		return err
	}
	buf := make([]byte, 4)
	xgb.Put32(buf, uint32(note))
	xproto.ChangeProperty(c.conn, xproto.PropModeReplace, win, typ, xproto.AtomAtom, 32, 1, buf)

	state, err := c.atom("_NET_WM_STATE")
	if err != nil {
		return err
	}
	taskbar, err := c.atom("_NET_WM_STATE_SKIP_TASKBAR")
	if err != nil {
		return err
	}
	pager, err := c.atom("_NET_WM_STATE_SKIP_PAGER")
	if err != nil {
		return err
	}
	ev := xproto.ClientMessageEvent{
		Format: 32,
		Window: win,
		Type:   state,
		Data:   xproto.ClientMessageDataUnionData32New(skipStateData(taskbar, pager)),
	}
	mask := uint32(xproto.EventMaskSubstructureRedirect | xproto.EventMaskSubstructureNotify)
	xproto.SendEvent(c.conn, false, c.root, mask, string(ev.Bytes()))
	_, err = xproto.GetInputFocus(c.conn).Reply()
	return err
}

func (c *Client) petWindow(title string) (xproto.Window, bool) {
	if c.petWin != 0 && strings.Contains(strings.ToLower(c.wmClass(c.petWin)), "catsitter") {
		return c.petWin, true
	}
	win, ok := c.findPet(title)
	if ok {
		c.petWin = win
	}
	return win, ok
}

func (c *Client) findPet(title string) (xproto.Window, bool) {
	var titleHit, classHit xproto.Window
	var haveTitle bool
	c.walk(c.root, 0, func(w xproto.Window) bool {
		if strings.Contains(strings.ToLower(c.wmClass(w)), "catsitter") {
			classHit = w
			return true
		}
		if !haveTitle && c.wmName(w) == title {
			titleHit = w
			haveTitle = true
		}
		return false
	})
	if classHit != 0 {
		return classHit, true
	}
	if haveTitle {
		return titleHit, true
	}
	return 0, false
}

func (c *Client) walk(w xproto.Window, depth int, visit func(xproto.Window) bool) bool {
	if depth > 16 {
		return false
	}
	if visit(w) {
		return true
	}
	tree, err := xproto.QueryTree(c.conn, w).Reply()
	if err != nil {
		return false
	}
	for _, child := range tree.Children {
		if c.walk(child, depth+1, visit) {
			return true
		}
	}
	return false
}

func (c *Client) wmName(w xproto.Window) string {
	if name := c.propString(w, "_NET_WM_NAME"); name != "" {
		return name
	}
	return c.propBytes(w, xproto.AtomWmName)
}

func (c *Client) wmClass(w xproto.Window) string {
	return c.propBytes(w, xproto.AtomWmClass)
}

func (c *Client) propString(w xproto.Window, name string) string {
	atom, err := c.atom(name)
	if err != nil {
		return ""
	}
	return c.prop(w, atom)
}

func (c *Client) propBytes(w xproto.Window, atom xproto.Atom) string {
	return c.prop(w, atom)
}

func (c *Client) prop(w xproto.Window, atom xproto.Atom) string {
	reply, err := xproto.GetProperty(c.conn, false, w, atom, xproto.AtomAny, 0, 128).Reply()
	if err != nil || len(reply.Value) == 0 {
		return ""
	}
	return strings.TrimRight(string(reply.Value), "\x00")
}

func (c *Client) atom(name string) (xproto.Atom, error) {
	if a, ok := c.atoms[name]; ok {
		return a, nil
	}
	reply, err := xproto.InternAtom(c.conn, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, err
	}
	if c.atoms == nil {
		c.atoms = map[string]xproto.Atom{}
	}
	c.atoms[name] = reply.Atom
	return reply.Atom, nil
}
