package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wanstu/wails-desktop-kit/atomicfile"
	kiticon "github.com/wanstu/wails-desktop-kit/icon"
)

func runIconPrepareWails(args []string) error {
	opts := kiticon.DefaultOptions()
	flags := flag.NewFlagSet("desktopkit icon prepare-wails", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	var input, desktopDir string
	var normalize bool
	var alpha int
	flags.StringVar(&input, "input", "", "source icon path")
	flags.StringVar(&desktopDir, "desktop-dir", ".", "directory containing wails.json/build")
	flags.BoolVar(&normalize, "normalize", false, "normalize source onto a square PNG before staging")
	flags.IntVar(&opts.CanvasSize, "canvas", opts.CanvasSize, "normalized output canvas size")
	flags.Float64Var(&opts.Fill, "fill", opts.Fill, "normalized artwork fill ratio")
	flags.BoolVar(&opts.TrimAlpha, "trim-alpha", opts.TrimAlpha, "trim transparent margins")
	flags.IntVar(&alpha, "alpha-threshold", int(opts.AlphaThreshold), "transparent trim alpha threshold 0-255")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if input == "" {
		return fmt.Errorf("--input is required")
	}
	if alpha < 0 || alpha > 255 {
		return fmt.Errorf("--alpha-threshold must be between 0 and 255")
	}
	opts.AlphaThreshold = uint8(alpha)

	buildIcon := filepath.Join(desktopDir, "build", "appicon.png")
	if err := os.MkdirAll(filepath.Dir(buildIcon), 0o755); err != nil {
		return err
	}
	if normalize {
		if err := kiticon.NormalizeFile(input, buildIcon, opts); err != nil {
			return err
		}
	} else {
		if err := atomicfile.Copy(input, buildIcon, 0o644); err != nil {
			return err
		}
	}
	windowsIcon := filepath.Join(desktopDir, "build", "windows", "icon.ico")
	if err := os.Remove(windowsIcon); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove generated Windows icon: %w", err)
	}
	fmt.Printf("prepared Wails icon: %s\n", buildIcon)
	return nil
}
