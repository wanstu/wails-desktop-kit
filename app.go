package desktopkit

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

// Hooks are application-owned lifecycle callbacks composed around the
// desktop-kit runtime.
type Hooks struct {
	Startup  func(context.Context)
	DomReady func(context.Context)
	Shutdown func(context.Context)
}

// Config describes a Wails desktop application shell.
type Config struct {
	ID     string
	Title  string
	Assets fs.FS
	Bind   []interface{}

	Launch LaunchOptions
	Window WindowConfig
	Tray   TrayConfig
	Hooks  Hooks

	// SingleInstance enables Wails' cross-platform single-instance lock.
	SingleInstance bool
}

// Run starts a Wails application using the common desktop-kit shell.
func Run(cfg Config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}

	window := cfg.Window.normalized()
	trayCfg := cfg.Tray.normalized(cfg.Title)
	controller := &Controller{}
	tray := newTrayManager(controller, trayCfg)

	hideOnClose := canHideCurrentPlatform(window.HidePolicy, trayCfg.Enabled && traySupported)
	startHidden := cfg.Launch.AutoStart && window.StartHiddenOnAutoStart && hideOnClose

	appOptions := &options.App{
		Title:             cfg.Title,
		Width:             window.Width,
		Height:            window.Height,
		MinWidth:          window.MinWidth,
		MinHeight:         window.MinHeight,
		StartHidden:       startHidden,
		HideWindowOnClose: hideOnClose,
		AssetServer:       &assetserver.Options{Assets: cfg.Assets},
		BackgroundColour: &options.RGBA{
			R: window.Background.R,
			G: window.Background.G,
			B: window.Background.B,
			A: window.Background.A,
		},
		OnStartup: func(ctx context.Context) {
			controller.setContext(ctx)
			tray.Startup(ctx)
			if cfg.Hooks.Startup != nil {
				cfg.Hooks.Startup(ctx)
			}
		},
		OnDomReady: func(ctx context.Context) {
			controller.setContext(ctx)
			if cfg.Hooks.DomReady != nil {
				cfg.Hooks.DomReady(ctx)
			}
			tray.DomReady(ctx)
		},
		OnShutdown: func(ctx context.Context) {
			tray.Shutdown(ctx)
			if cfg.Hooks.Shutdown != nil {
				cfg.Hooks.Shutdown(ctx)
			}
		},
		Bind: cfg.Bind,
	}

	if cfg.SingleInstance {
		appOptions.SingleInstanceLock = &options.SingleInstanceLock{
			UniqueId: cfg.ID,
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				controller.ShowWindow()
			},
		}
	}

	return wails.Run(appOptions)
}

func validateConfig(cfg Config) error {
	if cfg.ID == "" {
		return fmt.Errorf("desktop-kit: ID is required")
	}
	if cfg.Title == "" {
		return fmt.Errorf("desktop-kit: Title is required")
	}
	if cfg.Assets == nil {
		return fmt.Errorf("desktop-kit: Assets is required")
	}
	if cfg.Tray.Enabled && len(cfg.Tray.Icon) == 0 {
		return fmt.Errorf("desktop-kit: tray icon is required when tray is enabled")
	}
	if cfg.Tray.AutoStart != nil && !cfg.Tray.Enabled {
		return fmt.Errorf("desktop-kit: tray AutoStart requires tray to be enabled")
	}
	return nil
}
