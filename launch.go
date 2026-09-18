package desktopkit

import (
	"flag"
	"fmt"
	"io"
)

// LaunchOptions contains framework-owned desktop launch flags.
type LaunchOptions struct {
	AutoStart bool
}

// ParseLaunchOptions parses the standard desktop-kit launch flags.
//
// Applications with additional process modes may parse those modes first and
// call this function only for their desktop branch.
func ParseLaunchOptions(args []string) (LaunchOptions, error) {
	flags := flag.NewFlagSet("desktop", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	autostart := flags.Bool("autostart", false, "start after user login")
	if err := flags.Parse(args); err != nil {
		return LaunchOptions{}, err
	}
	if flags.NArg() != 0 {
		return LaunchOptions{}, fmt.Errorf("desktop accepts only --autostart")
	}
	return LaunchOptions{AutoStart: *autostart}, nil
}
