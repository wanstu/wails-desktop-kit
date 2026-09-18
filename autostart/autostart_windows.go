//go:build windows

package autostart

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const windowsRunKeyPath = "Software\\Microsoft\\Windows\\CurrentVersion\\Run"

func (m *Manager) supported() bool { return true }

func (m *Manager) command() (string, error) {
	executable, err := m.executable()
	if err != nil {
		return "", err
	}
	parts := []string{quoteWindowsArg(executable)}
	for _, arg := range m.args() {
		parts = append(parts, quoteWindowsArg(arg))
	}
	return strings.Join(parts, " "), nil
}

func (m *Manager) enabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, windowsRunKeyPath, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("autostart: open Windows Run key: %w", err)
	}
	defer key.Close()

	value, _, err := key.GetStringValue(m.config.ID)
	if errors.Is(err, registry.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("autostart: read Windows Run value: %w", err)
	}
	expected, err := m.command()
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(value), expected), nil
}

func (m *Manager) setEnabled(enabled bool) error {
	if !enabled {
		key, err := registry.OpenKey(registry.CURRENT_USER, windowsRunKeyPath, registry.SET_VALUE)
		if errors.Is(err, registry.ErrNotExist) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("autostart: open Windows Run key: %w", err)
		}
		defer key.Close()
		if err := key.DeleteValue(m.config.ID); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return fmt.Errorf("autostart: remove Windows Run value: %w", err)
		}
		return nil
	}

	command, err := m.command()
	if err != nil {
		return err
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, windowsRunKeyPath, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("autostart: create Windows Run key: %w", err)
	}
	defer key.Close()
	if err := key.SetStringValue(m.config.ID, command); err != nil {
		return fmt.Errorf("autostart: write Windows Run value: %w", err)
	}
	return nil
}

func quoteWindowsArg(arg string) string {
	if arg != "" && !strings.ContainsAny(arg, " \t\n\v\"") {
		return arg
	}
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for _, r := range arg {
		switch r {
		case '\\':
			slashes++
		case '"':
			b.WriteString(strings.Repeat("\\", slashes*2+1))
			b.WriteRune('"')
			slashes = 0
		default:
			if slashes > 0 {
				b.WriteString(strings.Repeat("\\", slashes))
				slashes = 0
			}
			b.WriteRune(r)
		}
	}
	if slashes > 0 {
		b.WriteString(strings.Repeat("\\", slashes*2))
	}
	b.WriteByte('"')
	return b.String()
}
