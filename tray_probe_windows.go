//go:build windows

package desktopkit

import (
	"context"
	"fmt"
	"github.com/gogpu/systray"
	"golang.org/x/sys/windows"
)

var postTrayThreadMessage = windows.NewLazySystemDLL("user32.dll").NewProc("PostThreadMessageW")

func trayLoopWake() func() {
	thread := windows.GetCurrentThreadId()
	return func() { postTrayThreadMessage.Call(uintptr(thread), 0x0012, 0, 0) }
}
func newTrayProbe(t *systray.SystemTray, _ string) (func(context.Context) error, func(), error) {
	return func(context.Context) error {
		_, _, w, h := t.Bounds()
		if w <= 0 || h <= 0 {
			return fmt.Errorf("Windows Shell did not confirm the tray icon")
		}
		return nil
	}, func() {}, nil
}
