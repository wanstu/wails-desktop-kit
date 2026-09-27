package main

import (
	"embed"
	"io/fs"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend
var frontend embed.FS

func main() {
	assets, err := fs.Sub(frontend, "frontend")
	if err != nil {
		panic(err)
	}
	if err := wails.Run(&options.App{
		Title:  "Desktop Kit Workflow Smoke",
		Width:  720,
		Height: 480,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
	}); err != nil {
		panic(err)
	}
}
