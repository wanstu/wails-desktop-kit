//go:build darwin

package autostart

import (
	"os"
	"strings"
	"testing"
)

func TestDarwinSetEnabledRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	manager, err := New(Config{
		ID:             "com.wanstu.example",
		DisplayName:    "Example",
		ExecutablePath: "/Applications/Example.app/Contents/MacOS/Example",
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
	path, err := manager.launchAgentPath()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "<string>--autostart</string>") {
		t.Fatalf("unexpected LaunchAgent: %s", data)
	}
	if err := manager.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("LaunchAgent still exists: %v", err)
	}
}
