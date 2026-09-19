package theme

import (
	"fmt"
	"regexp"
	"strings"
)

type Mode string

const (
	ModeLight  Mode = "light"
	ModeDark   Mode = "dark"
	ModeSystem Mode = "system"
)

var preferencePackNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type Preference struct {
	Mode Mode   `json:"mode"`
	Pack string `json:"pack"`
}

func DefaultPreference() Preference {
	return Preference{Mode: ModeSystem, Pack: "aurora"}
}

func ValidateMode(mode Mode) error {
	switch mode {
	case ModeLight, ModeDark, ModeSystem:
		return nil
	default:
		return fmt.Errorf("theme: invalid mode %q", mode)
	}
}

func ValidatePackName(pack string) error {
	pack = strings.TrimSpace(strings.ToLower(pack))
	if !preferencePackNamePattern.MatchString(pack) {
		return fmt.Errorf("theme: invalid pack %q", pack)
	}
	return nil
}

func NormalizePreference(value Preference) Preference {
	value.Mode = Mode(strings.TrimSpace(strings.ToLower(string(value.Mode))))
	value.Pack = strings.TrimSpace(strings.ToLower(value.Pack))
	defaults := DefaultPreference()
	if ValidateMode(value.Mode) != nil {
		value.Mode = defaults.Mode
	}
	if ValidatePackName(value.Pack) != nil {
		value.Pack = defaults.Pack
	}
	return value
}

func ValidatePreference(value Preference) error {
	if err := ValidateMode(value.Mode); err != nil {
		return err
	}
	return ValidatePackName(value.Pack)
}
