package desktopkit

import (
	"testing"
	"testing/fstest"
)

func TestValidateConfig(t *testing.T) {
	base := Config{
		ID:     "example",
		Title:  "Example",
		Assets: fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("ok")}},
	}
	if err := validateConfig(base); err != nil {
		t.Fatalf("valid config: %v", err)
	}

	withTray := base
	withTray.Tray = TrayConfig{Enabled: true}
	if err := validateConfig(withTray); err == nil {
		t.Fatal("expected missing tray icon error")
	}

	withTray.Tray.Icon = []byte{1}
	if err := validateConfig(withTray); err != nil {
		t.Fatalf("valid tray config: %v", err)
	}
}
