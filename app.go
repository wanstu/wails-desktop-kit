package desktopkit

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
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
	// BeforeClose returns true to cancel closing or quitting.
	BeforeClose func(context.Context) bool
	// TrayError reports initialization or runtime failure after restoring the window.
	TrayError func(context.Context, error)
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
	Theme  ThemeConfig
	Hooks  Hooks

	// SingleInstance enables Wails' cross-platform single-instance lock.
	SingleInstance bool
	// SecondInstance overrides the default show-window behavior and preserves launch arguments.
	SecondInstance func(*Controller, options.SecondInstanceData)
}

// Run starts a Wails application using the common desktop-kit shell.
func Run(cfg Config) error {
	if err := validateConfig(cfg); err != nil {
		return err
	}

	window := cfg.Window.normalized()
	trayCfg := cfg.Tray.normalized(cfg.Title)
	themeMiddleware := newThemeAssetMiddleware(cfg.Theme)
	hideOnClose := canHideCurrentPlatform(window.HidePolicy, trayCfg.Enabled && traySupported)
	startHidden := cfg.Launch.AutoStart && window.StartHiddenOnAutoStart && hideOnClose
	controller := newController(hideOnClose, startHidden)
	tray := newTrayManager(controller, trayCfg, func(err error) {
		if cfg.Hooks.TrayError != nil {
			cfg.Hooks.TrayError(controller.Context(), err)
		} else {
			controller.ShowError("托盘不可用，窗口已恢复", err)
		}
	})

	appOptions := &options.App{
		Title:             cfg.Title,
		Width:             window.Width,
		Height:            window.Height,
		MinWidth:          window.MinWidth,
		MinHeight:         window.MinHeight,
		StartHidden:       false,
		HideWindowOnClose: false,
		OnBeforeClose: func(ctx context.Context) bool {
			if controller.beforeClose(ctx, cfg.Hooks.BeforeClose) {
				return true
			}
			tray.Shutdown(ctx)
			return false
		},
		AssetServer: &assetserver.Options{Assets: cfg.Assets, Middleware: themeMiddleware},
		BackgroundColour: &options.RGBA{
			R: window.Background.R,
			G: window.Background.G,
			B: window.Background.B,
			A: window.Background.A,
		},
		OnStartup: func(ctx context.Context) {
			controller.setContext(ctx)
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
			controller.stop()
			if cfg.Hooks.Shutdown != nil {
				cfg.Hooks.Shutdown(ctx)
			}
		},
		Bind: cfg.Bind,
	}

	if cfg.SingleInstance {
		appOptions.SingleInstanceLock = &options.SingleInstanceLock{
			UniqueId: cfg.ID,
			OnSecondInstanceLaunch: func(data options.SecondInstanceData) {
				if cfg.SecondInstance != nil {
					cfg.SecondInstance(controller, data)
				} else {
					controller.ShowWindow()
				}
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
	if cfg.Tray.Enabled {
		image, err := png.DecodeConfig(bytes.NewReader(cfg.Tray.Icon))
		if err != nil || image.Width <= 0 || image.Height <= 0 || image.Width > 4096 || image.Height > 4096 {
			return fmt.Errorf("desktop-kit: tray icon must be a valid PNG no larger than 4096 pixels per side")
		}
		if _, err := png.Decode(bytes.NewReader(cfg.Tray.Icon)); err != nil {
			return fmt.Errorf("desktop-kit: decode tray PNG: %w", err)
		}
	}
	if cfg.Tray.AutoStart != nil && !cfg.Tray.Enabled {
		return fmt.Errorf("desktop-kit: tray AutoStart requires tray to be enabled")
	}
	return nil
}
