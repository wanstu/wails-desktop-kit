package desktopkit

import (
	"bytes"
	"image"
	"image/png"
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
	if err := validateConfig(withTray); err == nil {
		t.Fatal("expected invalid PNG error")
	}
	var icon bytes.Buffer
	if err := png.Encode(&icon, image.NewNRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	withTray.Tray.Icon = icon.Bytes()
	if err := validateConfig(withTray); err != nil {
		t.Fatalf("valid tray config: %v", err)
	}
}
