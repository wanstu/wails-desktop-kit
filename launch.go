package desktopkit

import (
	"flag"
	"fmt"
	"io"
	"strings"
)

// LaunchOptions contains framework-owned desktop launch flags.
type LaunchOptions struct {
	AutoStart bool
}

// SecondInstancePolicy controls the built-in response to a duplicate launch
// when Config.SecondInstance is not provided.
type SecondInstancePolicy string

const (
	// SecondInstanceWakeAlways preserves the historical Kit behavior.
	SecondInstanceWakeAlways SecondInstancePolicy = ""
	// SecondInstanceWakeManual wakes the existing window for manual launches,
	// but ignores login/autostart invocations.
	SecondInstanceWakeManual SecondInstancePolicy = "wake-manual"
	// SecondInstanceIgnore ignores every duplicate launch.
	SecondInstanceIgnore SecondInstancePolicy = "ignore"
)

// ParseLaunchOptions parses the standard desktop-kit launch flags.
func ParseLaunchOptions(args []string) (LaunchOptions, error) {
	return ParseLaunchOptionsWithAliases(args)
}

// ParseLaunchOptionsWithAliases also treats the supplied aliases as autostart
// flags. Aliases may be written as "minimized" or "--minimized".
func ParseLaunchOptionsWithAliases(args []string, autoStartAliases ...string) (LaunchOptions, error) {
	flags := flag.NewFlagSet("desktop", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var autostart bool
	flags.BoolVar(&autostart, "autostart", false, "start after user login")
	for _, alias := range autoStartAliases {
		name := strings.TrimLeft(strings.TrimSpace(alias), "-")
		if name == "" || name == "autostart" {
			continue
		}
		flags.BoolVar(&autostart, name, false, "start after user login")
	}
	if err := flags.Parse(args); err != nil {
		return LaunchOptions{}, err
	}
	if flags.NArg() != 0 {
		return LaunchOptions{}, fmt.Errorf("desktop accepts only autostart launch flags")
	}
	return LaunchOptions{AutoStart: autostart}, nil
}

// HasAutoStartArg reports whether args contain --autostart or one of the
// supplied aliases.
func HasAutoStartArg(args []string, aliases ...string) bool {
	accepted := map[string]struct{}{"--autostart": {}}
	for _, alias := range aliases {
		name := strings.TrimSpace(alias)
		if name == "" {
			continue
		}
		if !strings.HasPrefix(name, "--") {
			name = "--" + strings.TrimLeft(name, "-")
		}
		accepted[strings.ToLower(name)] = struct{}{}
	}
	for _, arg := range args {
		if _, ok := accepted[strings.ToLower(strings.TrimSpace(arg))]; ok {
			return true
		}
	}
	return false
}

func shouldWakeSecondInstance(policy SecondInstancePolicy, args, aliases []string) bool {
	switch policy {
	case SecondInstanceIgnore:
		return false
	case SecondInstanceWakeManual:
		return !HasAutoStartArg(args, aliases...)
	default:
		return true
	}
}
