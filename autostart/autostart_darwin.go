//go:build darwin

package autostart

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (m *Manager) supported() bool { return true }

func (m *Manager) launchAgentPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("autostart: resolve home directory: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", m.config.ID+".plist"), nil
}

func xmlText(value string) (string, error) {
	var out bytes.Buffer
	if err := xml.EscapeText(&out, []byte(value)); err != nil {
		return "", err
	}
	return out.String(), nil
}

func (m *Manager) launchAgentContent() ([]byte, error) {
	executable, err := m.executable()
	if err != nil {
		return nil, err
	}
	label, err := xmlText(m.config.ID)
	if err != nil {
		return nil, err
	}
	executable, err = xmlText(executable)
	if err != nil {
		return nil, err
	}

	var args strings.Builder
	args.WriteString("    <string>" + executable + "</string>\n")
	for _, arg := range m.args() {
		escaped, err := xmlText(arg)
		if err != nil {
			return nil, err
		}
		args.WriteString("    <string>" + escaped + "</string>\n")
	}

	content := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n" +
		"<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n" +
		"<plist version=\"1.0\">\n<dict>\n" +
		"  <key>Label</key>\n  <string>" + label + "</string>\n" +
		"  <key>ProgramArguments</key>\n  <array>\n" + args.String() + "  </array>\n" +
		"  <key>RunAtLoad</key>\n  <true/>\n" +
		"  <key>KeepAlive</key>\n  <false/>\n" +
		"</dict>\n</plist>\n"
	return []byte(content), nil
}

func (m *Manager) enabled() (bool, error) {
	path, err := m.launchAgentPath()
	if err != nil {
		return false, err
	}
	actual, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("autostart: read macOS LaunchAgent: %w", err)
	}
	expected, err := m.launchAgentContent()
	if err != nil {
		return false, err
	}
	return bytes.Equal(bytes.TrimSpace(actual), bytes.TrimSpace(expected)), nil
}

func (m *Manager) setEnabled(enabled bool) error {
	path, err := m.launchAgentPath()
	if err != nil {
		return err
	}
	if !enabled {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("autostart: remove macOS LaunchAgent: %w", err)
		}
		return nil
	}
	content, err := m.launchAgentContent()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("autostart: create macOS LaunchAgents directory: %w", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return fmt.Errorf("autostart: write macOS LaunchAgent: %w", err)
	}
	return nil
}
