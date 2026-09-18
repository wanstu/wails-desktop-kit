//go:build (!windows && !linux && !darwin) || (darwin && !cgo)

package desktopkit

import (
	"context"
	"fmt"
)

const traySupported = false

type unsupportedTray struct{}

func newTrayBackend([]byte, string) trayBackend { return &unsupportedTray{} }
func (*unsupportedTray) Run(context.Context, []nativeItem, func(int), func()) error {
	return fmt.Errorf("desktop-kit: native tray is unavailable on this platform/build")
}
func (*unsupportedTray) Update(int, bool, bool) {}
func (*unsupportedTray) Close()                 {}
