package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const maxAppIDLength = 64

// ValidateAppID validates the stable application identifier used in Kit-owned
// filesystem paths. The identifier must be a single path-safe component.
func ValidateAppID(appID string) error {
	if appID == "" {
		return errors.New("paths: app id is required")
	}
	if len(appID) > maxAppIDLength {
		return fmt.Errorf("paths: app id must be at most %d bytes", maxAppIDLength)
	}
	for i := 0; i < len(appID); i++ {
		c := appID[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_' || (c == '.' && i > 0) {
			continue
		}
		return fmt.Errorf("paths: app id %q contains unsupported character %q", appID, c)
	}
	if appID == "." || appID == ".." || strings.Contains(appID, "..") {
		return fmt.Errorf("paths: app id %q is not allowed", appID)
	}
	return nil
}

// ConfigRoot returns the Kit configuration root. XDG_CONFIG_HOME wins when it
// is set; otherwise Kit deliberately standardizes on ~/.config on every OS.
func ConfigRoot() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); configured != "" {
		if !filepath.IsAbs(configured) {
			return "", fmt.Errorf("paths: XDG_CONFIG_HOME must be absolute: %q", configured)
		}
		return filepath.Clean(configured), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("paths: resolve user home: %w", err)
	}
	if strings.TrimSpace(home) == "" {
		return "", errors.New("paths: user home is empty")
	}
	return filepath.Join(home, ".config"), nil
}

// ConfigDir returns ~/.config/<appID> (or $XDG_CONFIG_HOME/<appID>).
func ConfigDir(appID string) (string, error) {
	if err := ValidateAppID(appID); err != nil {
		return "", err
	}
	root, err := ConfigRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, appID), nil
}

// EnsureConfigDir creates and returns the app configuration directory.
func EnsureConfigDir(appID string) (string, error) {
	dir, err := ConfigDir(appID)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("paths: create config dir: %w", err)
	}
	return dir, nil
}
