package main

import (
	"flag"
	"fmt"
	"os"

	kiticon "github.com/wanstu/wails-desktop-kit/icon"
	kitpackaging "github.com/wanstu/wails-desktop-kit/packaging"
)

const usage = `desktopkit - Wails Desktop Kit engineering helper

Usage:
  desktopkit icon [normalize options]
  desktopkit icon normalize [options]
  desktopkit icon generate [options]
  desktopkit package linux [options]

Commands:
  icon normalize    Normalize a PNG/JPEG/GIF onto a square transparent PNG canvas.
  icon generate     Generate a deterministic Kit family icon without system fonts.
  package linux     Stage Linux release artifacts (raw, deb, tar.gz) and SHA256 files.

Existing "desktopkit icon --input ... --output ..." remains compatible.
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "desktopkit:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		fmt.Print(usage)
		return nil
	}
	switch args[0] {
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	case "icon":
		return runIcon(args[1:])
	case "package":
		return runPackage(args[1:])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runIcon(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "generate":
			return runIconGenerate(args[1:])
		case "normalize":
			return runIconNormalize(args[1:])
		case "help", "-h", "--help":
			fmt.Print(usage)
			return nil
		}
	}
	return runIconNormalize(args)
}

func runIconNormalize(args []string) error {
	opts := kiticon.DefaultOptions()
	flags := flag.NewFlagSet("desktopkit icon normalize", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	var input string
	var output string
	var alpha int

	flags.StringVar(&input, "input", "", "source PNG/JPEG/GIF path")
	flags.StringVar(&output, "output", "", "destination PNG path")
	flags.IntVar(&opts.CanvasSize, "canvas", opts.CanvasSize, "square output size in pixels")
	flags.Float64Var(&opts.Fill, "fill", opts.Fill, "visible artwork fill ratio (0,1]")
	flags.BoolVar(&opts.TrimAlpha, "trim-alpha", opts.TrimAlpha, "trim transparent margins before fitting")
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
	if output == "" {
		return fmt.Errorf("--output is required")
	}
	if alpha < 0 || alpha > 255 {
		return fmt.Errorf("--alpha-threshold must be between 0 and 255")
	}
	opts.AlphaThreshold = uint8(alpha)

	if err := kiticon.NormalizeFile(input, output, opts); err != nil {
		return err
	}
	fmt.Printf("normalized icon: %s -> %s\n", input, output)
	return nil
}

func runIconGenerate(args []string) error {
	opts := kiticon.DefaultBadgeOptions()
	flags := flag.NewFlagSet("desktopkit icon generate", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	var output string
	var background string
	var foreground string

	flags.StringVar(&output, "output", "", "destination PNG path")
	flags.IntVar(&opts.Size, "size", opts.Size, "square output size in pixels (16-1024)")
	flags.IntVar(&opts.Inset, "inset", opts.Inset, "transparent outer inset in pixels")
	flags.IntVar(&opts.Radius, "radius", opts.Radius, "rounded-square corner radius in pixels")
	flags.StringVar(&opts.Symbol, "symbol", opts.Symbol, "symbol: terminal or monogram")
	flags.StringVar(&opts.Text, "text", "", "single ASCII letter/digit for monogram")
	flags.StringVar(&background, "background", "#2463EB", "background color #RRGGBB or #RRGGBBAA")
	flags.StringVar(&foreground, "foreground", "#FFFFFF", "foreground color #RRGGBB or #RRGGBBAA")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if output == "" {
		return fmt.Errorf("--output is required")
	}

	bg, err := kiticon.ParseHexColor(background)
	if err != nil {
		return fmt.Errorf("--background: %w", err)
	}
	fg, err := kiticon.ParseHexColor(foreground)
	if err != nil {
		return fmt.Errorf("--foreground: %w", err)
	}
	opts.Background = bg
	opts.Foreground = fg

	if err := kiticon.GenerateBadgeFile(output, opts); err != nil {
		return err
	}
	fmt.Printf("generated icon: %s (%s)\n", output, opts.Symbol)
	return nil
}

func runPackage(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("package target is required (supported: linux)")
	}
	switch args[0] {
	case "linux":
		return runPackageLinux(args[1:])
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	default:
		return fmt.Errorf("unsupported package target %q", args[0])
	}
}

func runPackageLinux(args []string) error {
	flags := flag.NewFlagSet("desktopkit package linux", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	var request kitpackaging.LinuxRequest
	var formats string

	flags.StringVar(&request.Input, "input", "", "built Linux executable path")
	flags.StringVar(&request.OutputDir, "dist", "dist", "output directory")
	flags.StringVar(&request.AppName, "app-name", "", "installed executable name")
	flags.StringVar(&request.AssetBase, "asset-base", "", "release filename prefix, without platform suffix")
	flags.StringVar(&request.PackageName, "package-name", "", "Debian package / desktop asset name (defaults from app-name)")
	flags.StringVar(&request.PackageVersion, "package-version", "0.0.0", "package version (Debian version syntax, usually tag without leading v)")
	flags.StringVar(&request.Architecture, "arch", "amd64", "Linux architecture: amd64 or arm64")
	flags.StringVar(&request.Description, "description", "Wails desktop application", "package description")
	flags.StringVar(&request.Maintainer, "maintainer", "unknown", "package maintainer")
	flags.StringVar(&request.Section, "section", "utils", "Debian section")
	flags.StringVar(&request.Priority, "priority", "optional", "Debian priority")
	flags.StringVar(&request.Depends, "depends", "", "Debian Depends value")
	flags.StringVar(&request.DesktopFile, "desktop-file", "", "optional .desktop file included in deb/tar.gz")
	flags.StringVar(&request.IconFile, "icon", "", "optional app icon included in deb/tar.gz")
	flags.StringVar(&formats, "formats", "raw", "comma-separated formats: raw,deb,tar.gz")

	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if request.Input == "" {
		return fmt.Errorf("--input is required")
	}
	if request.AppName == "" {
		return fmt.Errorf("--app-name is required")
	}
	if request.AssetBase == "" {
		request.AssetBase = request.AppName
	}

	parsed, err := kitpackaging.ParseFormats(formats)
	if err != nil {
		return err
	}
	request.Formats = parsed

	artifacts, err := kitpackaging.PackageLinux(request)
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		fmt.Printf("%s  %s  %s\n", artifact.Format, artifact.SHA256, artifact.Path)
	}
	return nil
}
