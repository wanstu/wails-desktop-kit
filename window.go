package desktopkit

import "runtime"

// HidePolicy controls whether closing the main window may hide it to the tray.
type HidePolicy string

const (
	// HideSafe hides on platforms where desktop-kit can reliably recover the
	// window from the tray. Linux currently stays visible because a
	// StatusNotifier host is not guaranteed to exist.
	HideSafe HidePolicy = "safe"
	// HideAlways always enables hide-on-close when the tray is enabled.
	HideAlways HidePolicy = "always"
	// HideNever never hides the window on close.
	HideNever HidePolicy = "never"
)

// Color is an RGBA application background color.
type Color struct {
	R uint8
	G uint8
	B uint8
	A uint8
}

// WindowConfig describes the common Wails window shell.
type WindowConfig struct {
	Width     int
	Height    int
	MinWidth  int
	MinHeight int

	HidePolicy             HidePolicy
	StartHiddenOnAutoStart bool
	Background             Color
}

// DefaultWindowConfig returns the shared desktop-kit window defaults.
func DefaultWindowConfig() WindowConfig {
	return WindowConfig{
		Width:                  1024,
		Height:                 720,
		MinWidth:               720,
		MinHeight:              520,
		HidePolicy:             HideAlways,
		StartHiddenOnAutoStart: true,
		Background:             Color{R: 246, G: 247, B: 249, A: 1},
	}
}

func (c WindowConfig) normalized() WindowConfig {
	d := DefaultWindowConfig()
	if c.Width <= 0 {
		c.Width = d.Width
	}
	if c.Height <= 0 {
		c.Height = d.Height
	}
	if c.MinWidth <= 0 {
		c.MinWidth = d.MinWidth
	}
	if c.MinHeight <= 0 {
		c.MinHeight = d.MinHeight
	}
	if c.HidePolicy == "" {
		c.HidePolicy = d.HidePolicy
	}
	if c.Background.A == 0 && c.Background.R == 0 && c.Background.G == 0 && c.Background.B == 0 {
		c.Background = d.Background
	}
	return c
}

// CanHideToTray reports whether a platform should allow the main window to
// become unreachable except through the tray.
func CanHideToTray(policy HidePolicy, trayEnabled bool, goos string) bool {
	if !trayEnabled {
		return false
	}
	switch policy {
	case HideAlways:
		return true
	case HideNever:
		return false
	case HideSafe, "":
		return goos == "windows" || goos == "darwin"
	default:
		return false
	}
}

func canHideCurrentPlatform(policy HidePolicy, trayEnabled bool) bool {
	return CanHideToTray(policy, trayEnabled, runtime.GOOS)
}
