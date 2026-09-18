//go:build linux

package autostart

import (
	"os"
	"strings"
	"testing"
)

func TestLinuxSetEnabledRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	manager, err := New(Config{
		ID:             "com.wanstu.example",
		DisplayName:    "Example",
		Comment:        "Example desktop app",
		ExecutablePath: "/opt/Example App/example",
		Arguments:      []string{"--autostart"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	enabled, err := manager.Enabled()
	if err != nil {
		t.Fatal(err)
	}
	if !enabled {
		t.Fatal("Enabled() = false after SetEnabled(true)")
	}
	path, err := manager.linuxPath()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `Exec="/opt/Example App/example" "--autostart"`) {
		t.Fatalf("unexpected desktop entry: %s", data)
	}
	if err := manager.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("desktop entry still exists: %v", err)
	}
}
