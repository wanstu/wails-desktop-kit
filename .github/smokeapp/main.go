// This executable verifies that the native tray shares Wails' main loop and
// can be created and removed while a real WebView is running.
package main

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"sync"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	desktopkit "github.com/wanstu/wails-desktop-kit"
)

//go:embed index.html
var assets embed.FS

func main() {
	var icon bytes.Buffer
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 30, G: 140, B: 90, A: 255})
		}
	}
	if err := png.Encode(&icon, img); err != nil {
		panic(err)
	}
	var mu sync.Mutex
	var trayErr error
	domReady, shutdown := false, false
	window := desktopkit.DefaultWindowConfig()
	window.HidePolicy = desktopkit.HideNever
	go func() { time.Sleep(30 * time.Second); fmt.Fprintln(os.Stderr, "smoke timeout"); os.Exit(2) }()
	err := desktopkit.Run(desktopkit.Config{
		ID: "desktopkit-runtime-smoke", Title: "Desktop Kit smoke", Assets: assets, Window: window,
		Tray: desktopkit.TrayConfig{Enabled: true, Icon: icon.Bytes()},
		Hooks: desktopkit.Hooks{
			DomReady: func(ctx context.Context) {
				mu.Lock()
				domReady = true
				mu.Unlock()
				go func() { time.Sleep(3 * time.Second); wailsruntime.Quit(ctx) }()
			},
			TrayError: func(ctx context.Context, err error) { mu.Lock(); trayErr = err; mu.Unlock(); wailsruntime.Quit(ctx) },
			Shutdown:  func(context.Context) { mu.Lock(); shutdown = true; mu.Unlock() },
		},
	})
	mu.Lock()
	defer mu.Unlock()
	if err != nil || trayErr != nil || !domReady || !shutdown {
		fmt.Fprintf(os.Stderr, "smoke failed: run=%v tray=%v dom=%t shutdown=%t\n", err, trayErr, domReady, shutdown)
		os.Exit(1)
	}
	fmt.Println("Wails WebView + native tray startup/shutdown passed")
}
