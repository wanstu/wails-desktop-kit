//go:build !windows && !linux && !darwin

package desktopkit

import "context"

const traySupported = false

type trayManager struct{}

func newTrayManager(*Controller, TrayConfig) *trayManager { return &trayManager{} }
func (*trayManager) Startup(context.Context)              {}
func (*trayManager) DomReady(context.Context)             {}
func (*trayManager) Shutdown(context.Context)             {}
