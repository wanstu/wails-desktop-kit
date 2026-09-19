//go:build linux

package desktopkit

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/godbus/dbus/v5"
	"github.com/wanstu/systray"
)

func trayLoopWake() func() { return func() {} }
func newTrayProbe(_ *systray.SystemTray, title string) (func(context.Context) error, func(), error) {
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, nil, fmt.Errorf("connect tray availability probe: %w", err)
	}
	prefix := fmt.Sprintf("org.kde.StatusNotifierItem-%d-", os.Getpid())
	probe := func(ctx context.Context) error {
		var names []string
		if err := conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
			return err
		}
		for _, name := range names {
			if !strings.HasPrefix(name, prefix) {
				continue
			}
			obj := conn.Object(name, dbus.ObjectPath("/StatusNotifierItem"))
			var props map[string]dbus.Variant
			if err := obj.CallWithContext(ctx, "org.freedesktop.DBus.Properties.GetAll", 0, "org.kde.StatusNotifierItem").Store(&props); err != nil {
				continue
			}
			if props["Title"].Value() == title && props["Status"].Value() == "Active" {
				// This confirms a live exported item, not a visible host. HideSafe still
				// refuses automatic hiding on Linux; HideAlways is an explicit override.
				return nil
			}
		}
		return fmt.Errorf("Linux StatusNotifierItem was not exported successfully")
	}
	return probe, func() { _ = conn.Close() }, nil
}
