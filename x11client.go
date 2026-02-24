package main

import (
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/sirupsen/logrus"
)

const iconicState = 3

type X11Client struct {
	conn  *xgb.Conn
	root  xproto.Window
	atoms struct {
		netClientList   xproto.Atom
		netWmDesktop    xproto.Atom
		netWmName       xproto.Atom
		utf8String      xproto.Atom
		netActiveWindow xproto.Atom
		wmChangeState   xproto.Atom
	}
}

func NewX11Client() (*X11Client, error) {
	conn, err := xgb.NewConn()
	if err != nil {
		return nil, fmt.Errorf("xgb connect: %w", err)
	}

	setup := xproto.Setup(conn)
	root := setup.DefaultScreen(conn).Root

	c := &X11Client{conn: conn, root: root}
	if err := c.cacheAtoms(); err != nil {
		conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *X11Client) Close() {
	c.conn.Close()
}

// ListWindows returns instance-name -> windows, matching the old getRunningInstances() output.
func (c *X11Client) ListWindows() map[string][]WmCtrlInstance {
	data, err := c.getProp(c.root, c.atoms.netClientList, xproto.AtomWindow, 512)
	if err != nil {
		logrus.Error("get _NET_CLIENT_LIST:", err)
		return nil
	}

	apps := make(map[string][]WmCtrlInstance)

	for i := 0; i+3 < len(data); i += 4 {
		wid := xproto.Window(binary.LittleEndian.Uint32(data[i:]))

		// Skip windows on desktop -1 (e.g. always-on-top dock windows)
		desktopData, err := c.getProp(wid, c.atoms.netWmDesktop, xproto.AtomCardinal, 1)
		if err != nil || len(desktopData) < 4 {
			continue
		}
		if desktop := int32(binary.LittleEndian.Uint32(desktopData)); desktop == -1 {
			continue
		}

		// WM_CLASS = "instance\0class\0"; first part is the instance name
		classData, err := c.getProp(wid, xproto.AtomWmClass, xproto.AtomString, 64)
		if err != nil || len(classData) == 0 {
			continue
		}
		parts := strings.SplitN(string(classData), "\x00", 3)
		if len(parts) < 1 || parts[0] == "" {
			continue
		}
		instanceName := strings.ToLower(parts[0])

		// Prefer _NET_WM_NAME (UTF-8), fall back to WM_NAME
		title := ""
		if td, err := c.getProp(wid, c.atoms.netWmName, c.atoms.utf8String, 128); err == nil && len(td) > 0 {
			title = string(td)
		} else if td, err := c.getProp(wid, xproto.AtomWmName, xproto.AtomString, 128); err == nil && len(td) > 0 {
			title = string(td)
		}

		apps[instanceName] = append(apps[instanceName], WmCtrlInstance{
			WindowId: fmt.Sprintf("0x%08x", uint32(wid)),
			Name:     title,
		})
	}

	return apps
}

// FocusWindow raises and activates the window.
// source=2 (pager) is used so KWin bypasses focus-stealing prevention,
// which causes random failures with source=1 + timestamp=0.
func (c *X11Client) FocusWindow(id string) error {
	wid, err := parseWindowID(id)
	if err != nil {
		return err
	}

	// Map the window first in case it is iconified
	xproto.MapWindow(c.conn, wid)

	// Ask the WM to activate: source=2 (pager), timestamp=0, current active=0
	return c.sendClientMessage(wid, c.atoms.netActiveWindow,
		xproto.EventMaskSubstructureNotify|xproto.EventMaskSubstructureRedirect,
		[5]uint32{2, 0, 0, 0, 0})
}

// MinimizeWindow iconifies the window via WM_CHANGE_STATE.
func (c *X11Client) MinimizeWindow(id string) error {
	wid, err := parseWindowID(id)
	if err != nil {
		return err
	}
	return c.sendClientMessage(wid, c.atoms.wmChangeState,
		xproto.EventMaskSubstructureNotify|xproto.EventMaskSubstructureRedirect,
		[5]uint32{iconicState, 0, 0, 0, 0})
}

func (c *X11Client) sendClientMessage(win xproto.Window, msgType xproto.Atom, mask uint32, data [5]uint32) error {
	// Manually build the 32-byte ClientMessage event (code 33, format 32).
	buf := make([]byte, 32)
	buf[0] = 33 // ClientMessage event code
	buf[1] = 32 // data format: 32-bit
	// buf[2:4] sequence = 0
	binary.LittleEndian.PutUint32(buf[4:], uint32(win))
	binary.LittleEndian.PutUint32(buf[8:], uint32(msgType))
	for i, v := range data {
		binary.LittleEndian.PutUint32(buf[12+i*4:], v)
	}

	err := xproto.SendEventChecked(c.conn, false, c.root, mask, string(buf)).Check()
	if err != nil {
		logrus.Errorf("sendClientMessage to 0x%x: %v", uint32(win), err)
	}
	return err
}

func parseWindowID(id string) (xproto.Window, error) {
	var wid uint32
	if _, err := fmt.Sscanf(id, "0x%x", &wid); err != nil {
		if _, err := fmt.Sscanf(id, "%d", &wid); err != nil {
			return 0, fmt.Errorf("invalid window ID %q", id)
		}
	}
	return xproto.Window(wid), nil
}

func (c *X11Client) internAtom(name string) (xproto.Atom, error) {
	reply, err := xproto.InternAtom(c.conn, false, uint16(len(name)), name).Reply()
	if err != nil {
		return 0, fmt.Errorf("intern atom %q: %w", name, err)
	}
	return reply.Atom, nil
}

func (c *X11Client) cacheAtoms() error {
	entries := []struct {
		ptr  *xproto.Atom
		name string
	}{
		{&c.atoms.netClientList, "_NET_CLIENT_LIST"},
		{&c.atoms.netWmDesktop, "_NET_WM_DESKTOP"},
		{&c.atoms.netWmName, "_NET_WM_NAME"},
		{&c.atoms.utf8String, "UTF8_STRING"},
		{&c.atoms.netActiveWindow, "_NET_ACTIVE_WINDOW"},
		{&c.atoms.wmChangeState, "WM_CHANGE_STATE"},
	}
	for _, e := range entries {
		atom, err := c.internAtom(e.name)
		if err != nil {
			return err
		}
		*e.ptr = atom
	}
	return nil
}

func (c *X11Client) getProp(win xproto.Window, atom, typ xproto.Atom, longLen uint32) ([]byte, error) {
	reply, err := xproto.GetProperty(c.conn, false, win, atom, typ, 0, longLen).Reply()
	if err != nil {
		return nil, err
	}
	return reply.Value, nil
}
