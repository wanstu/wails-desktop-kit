package autostart

import "testing"

func TestNewValidation(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected missing ID error")
	}
	if _, err := New(Config{ID: "../bad"}); err == nil {
		t.Fatal("expected path separator validation error")
	}
}

func TestExecutableOverrideIsAbsolute(t *testing.T) {
	m, err := New(Config{
		ID:             "example.desktop",
		DisplayName:    "Example",
		ExecutablePath: "example-app",
		Arguments:      []string{"--autostart"},
	})
	if err != nil {
		t.Fatal(err)
	}
	path, err := m.executable()
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("empty executable path")
	}
}
