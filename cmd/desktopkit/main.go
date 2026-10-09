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
  desktopkit icon prepare-wails [options]
  desktopkit package linux [options]
  desktopkit package windows [options]
  desktopkit buildmeta prepare [options]
  desktopkit doctor [options]
  desktopkit upgrade [options]

Commands:
  icon normalize    Normalize a PNG/JPEG/GIF onto a square transparent PNG canvas.
  icon generate     Generate a deterministic Kit family icon without system fonts.
  icon prepare-wails Prepare build/appicon.png and invalidate generated Windows ICO.
  package linux     Stage raw/deb/tar.gz and optional systemd service packages, with SHA256.
  package windows   Build an NSIS installer from an already-built Windows executable.
  buildmeta prepare Inject release version into Wails metadata and frontend build info.
  doctor            Inspect Kit/Wails/workflow and fixed build-script Kit CLI versions.
  upgrade           Update Kit module, workflow, and fixed build-script CLI references.

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
	case "buildmeta":
		return runBuildMeta(args[1:])
	case "doctor":
		return runDoctor(args[1:])
	case "upgrade":
		return runUpgrade(args[1:])
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
		case "prepare-wails":
			return runIconPrepareWails(args[1:])
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
		return fmt.Errorf("package target is required (supported: linux, windows)")
	}
	switch args[0] {
	case "linux":
		return runPackageLinux(args[1:])
	case "windows":
		return runPackageWindows(args[1:])
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
	var extraBins stringValues
	var serviceArgs stringValues
	var serviceEnv stringValues
	var systemd bool
	var service kitpackaging.LinuxSystemdService

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
	flags.StringVar(&request.CopyrightFile, "copyright-file", "", "optional Debian copyright/license text included under /usr/share/doc/<package>/copyright")
	flags.StringVar(&request.AppStreamFile, "appstream-file", "", "optional application AppStream .metainfo.xml, declares product license if known")
	flags.StringVar(&request.DesktopFile, "desktop-file", "", "optional .desktop file included in deb/tar.gz")
	flags.StringVar(&request.IconFile, "icon", "", "optional app icon included in deb/tar.gz")
	flags.Var(&extraBins, "extra-bin", "additional installed executable as name=path; may be repeated")
	flags.StringVar(&formats, "formats", "raw", "comma-separated formats: raw,deb,tar.gz")
	flags.BoolVar(&systemd, "systemd", false, "enable systemd service packaging for Linux deb")
	flags.StringVar(&service.Name, "service-name", "", "systemd unit name, defaults to package-name")
	flags.StringVar(&service.Description, "service-description", "", "systemd unit description")
	flags.StringVar(&service.User, "service-user", "", "unprivileged system user created by the package")
	flags.StringVar(&service.Group, "service-group", "", "system group created by the package")
	flags.StringVar(&service.DataDir, "service-data-dir", "", "persistent directory under /var/lib")
	flags.StringVar(&service.EnvironmentFile, "service-environment-file", "", "optional /etc/default/<service> configuration file")
	flags.Var(&serviceEnv, "service-env", "default environment variable NAME=value; repeatable")
	flags.Var(&serviceArgs, "service-arg", "one ExecStart argument; repeat to supply all startup flags")

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
	if systemd {
		service.Args = append([]string(nil), serviceArgs...)
		service.Environment = append([]string(nil), serviceEnv...)
		request.Systemd = &service
	}
	request.ExtraBinaries, err = parseExtraBinaries(extraBins)
	if err != nil {
		return err
	}

	artifacts, err := kitpackaging.PackageLinux(request)
	if err != nil {
		return err
	}
	for _, artifact := range artifacts {
		fmt.Printf("%s  %s  %s\n", artifact.Format, artifact.SHA256, artifact.Path)
	}
	return nil
}

func runPackageWindows(args []string) error {
	flags := flag.NewFlagSet("desktopkit package windows", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)

	var request kitpackaging.WindowsRequest
	flags.StringVar(&request.Input, "input", "", "already-built Windows executable path")
	flags.StringVar(&request.OutputDir, "dist", "dist", "output directory")
	flags.StringVar(&request.AppName, "app-name", "", "installed executable name, without .exe")
	flags.StringVar(&request.AssetBase, "asset-base", "", "release filename prefix, without platform suffix")
	flags.StringVar(&request.PackageVersion, "package-version", "0.0.0", "display version, usually tag without leading v")
	flags.StringVar(&request.Architecture, "arch", "amd64", "Windows architecture: amd64 or arm64")
	flags.StringVar(&request.InstallScope, "install-scope", "user", "install scope: user or machine")
	flags.StringVar(&request.ProductName, "product-name", "", "display product name (defaults to app-name)")
	flags.StringVar(&request.Publisher, "publisher", "Unknown", "Windows publisher/company display name")
	flags.StringVar(&request.AppID, "app-id", "", "stable uninstall registry id (defaults to app-name)")
	flags.StringVar(&request.IconFile, "icon", "", "optional .ico installer icon")
	flags.BoolVar(&request.StartMenuShortcut, "start-menu-shortcut", true, "create Start Menu shortcuts")
	flags.BoolVar(&request.DesktopShortcut, "desktop-shortcut", false, "create a desktop shortcut")
	flags.StringVar(&request.NSISPath, "nsis", "", "makensis executable path (defaults to PATH lookup)")

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

	artifact, err := kitpackaging.PackageWindows(request)
	if err != nil {
		return err
	}
	fmt.Printf("%s  %s  %s\n", artifact.Format, artifact.SHA256, artifact.Path)
	return nil
}
