//go:build linux

package autostart

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (m *Manager) supported() bool { return true }

func (m *Manager) linuxPath() (string, error) {
	configHome := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if configHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("autostart: resolve home directory: %w", err)
		}
		configHome = filepath.Join(home, ".config")
	}
	return filepath.Join(configHome, "autostart", m.config.ID+".desktop"), nil
}

func (m *Manager) linuxContent() ([]byte, error) {
	executable, err := m.executable()
	if err != nil {
		return nil, err
	}
	parts := []string{quoteDesktopExecArg(executable)}
	for _, arg := range m.args() {
		parts = append(parts, quoteDesktopExecArg(arg))
	}
	comment := strings.TrimSpace(m.config.Comment)
	if comment == "" {
		comment = m.config.DisplayName
	}
	content := fmt.Sprintf(
		"[Desktop Entry]\nType=Application\nVersion=1.0\nName=%s\nComment=%s\nExec=%s\nTerminal=false\nX-GNOME-Autostart-enabled=true\n",
		escapeDesktopValue(m.config.DisplayName),
		escapeDesktopValue(comment),
		strings.Join(parts, " "),
	)
	return []byte(content), nil
}

func (m *Manager) enabled() (bool, error) {
	path, err := m.linuxPath()
	if err != nil {
		return false, err
	}
	actual, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("autostart: read Linux desktop entry: %w", err)
	}
	expected, err := m.linuxContent()
	if err != nil {
		return false, err
	}
	return bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)), nil
}

func (m *Manager) setEnabled(enabled bool) error {
	path, err := m.linuxPath()
	if err != nil {
		return err
	}
	if !enabled {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("autostart: remove Linux desktop entry: %w", err)
		}
		return nil
	}
	content, err := m.linuxContent()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("autostart: create Linux autostart directory: %w", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return fmt.Errorf("autostart: write Linux desktop entry: %w", err)
	}
	return nil
}

func quoteDesktopExecArg(value string) string {
	value = strings.ReplaceAll(value, "%", "%%")
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"`", "\\`",
		"$", "\\$",
	)
	return "\"" + replacer.Replace(value) + "\""
}

func escapeDesktopValue(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\n", "\\n")
	return value
}
