//go:build !windows && !linux && !darwin

package autostart

func (m *Manager) supported() bool        { return false }
func (m *Manager) enabled() (bool, error) { return false, nil }
func (m *Manager) setEnabled(bool) error  { return nil }
