package autostart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	ID             string
	DisplayName    string
	Comment        string
	Arguments      []string
	ExecutablePath string
}

type Manager struct {
	config Config
}

func New(config Config) (*Manager, error) {
	config.ID = strings.TrimSpace(config.ID)
	config.DisplayName = strings.TrimSpace(config.DisplayName)
	if config.ID == "" {
		return nil, fmt.Errorf("autostart: ID is required")
	}
	if strings.ContainsAny(config.ID, "/\\") {
		return nil, fmt.Errorf("autostart: ID must not contain path separators")
	}
	if config.DisplayName == "" {
		config.DisplayName = config.ID
	}
	return &Manager{config: config}, nil
}

func (m *Manager) Supported() bool {
	return m != nil && m.supported()
}

func (m *Manager) Enabled() (bool, error) {
	if m == nil {
		return false, fmt.Errorf("autostart: nil manager")
	}
	if !m.supported() {
		return false, nil
	}
	return m.enabled()
}

func (m *Manager) SetEnabled(enabled bool) error {
	if m == nil {
		return fmt.Errorf("autostart: nil manager")
	}
	if !m.supported() {
		return fmt.Errorf("autostart: unsupported platform")
	}
	return m.setEnabled(enabled)
}

func (m *Manager) executable() (string, error) {
	path := strings.TrimSpace(m.config.ExecutablePath)
	if path == "" {
		var err error
		path, err = os.Executable()
		if err != nil {
			return "", fmt.Errorf("autostart: resolve executable: %w", err)
		}
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("autostart: resolve executable path: %w", err)
	}
	return filepath.Clean(absolute), nil
}

func (m *Manager) args() []string {
	return append([]string(nil), m.config.Arguments...)
}
