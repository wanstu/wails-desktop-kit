package main

import (
	"flag"
	"fmt"
	"os"

	kiticon "github.com/wanstu/wails-desktop-kit/icon"
)

const usage = "desktopkit - Wails Desktop Kit engineering helper\n\nUsage:\n  desktopkit icon [options]\n\nCommands:\n  icon    Normalize one application/tray/window image onto a square PNG canvas.\n"

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
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runIcon(args []string) error {
	opts := kiticon.DefaultOptions()
	flags := flag.NewFlagSet("desktopkit icon", flag.ContinueOnError)
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
