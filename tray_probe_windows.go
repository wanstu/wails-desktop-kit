//go:build windows

package desktopkit

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/gogpu/systray"
	"golang.org/x/sys/windows"
)

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	postTrayThreadMessage = user32.NewProc("PostThreadMessageW")
	findWindowW           = user32.NewProc("FindWindowW")
)

func trayLoopWake() func() {
	thread := windows.GetCurrentThreadId()
	return func() { postTrayThreadMessage.Call(uintptr(thread), 0x0012, 0, 0) }
}

func windowsTrayShellAvailable() bool {
	className, err := windows.UTF16PtrFromString("Shell_TrayWnd")
	if err != nil {
		return false
	}
	hwnd, _, _ := findWindowW.Call(uintptr(unsafe.Pointer(className)), 0)
	return hwnd != 0
}

func latchingTrayProbe(check func() error) func(context.Context) error {
	confirmed := false
	return func(context.Context) error {
		if confirmed {
			return nil
		}
		if err := check(); err != nil {
			return err
		}
		confirmed = true
		return nil
	}
}

func newTrayProbe(_ *systray.SystemTray, _ string) (func(context.Context) error, func(), error) {
	// Bounds() is not a reliable liveness signal on Windows 11: an icon in
	// the notification-area overflow can temporarily report a zero-sized
	// rectangle. Treating that as permanent tray loss removes our own icon
	// and forces a hidden/autostart window visible. Confirm the Explorer tray
	// host once at startup instead; after that the native event loop owns
	// liveness.
	probe := latchingTrayProbe(func() error {
		if !windowsTrayShellAvailable() {
			return fmt.Errorf("Windows Shell tray host is unavailable")
		}
		return nil
	})
	return probe, func() {}, nil
}
