package ui

import (
	"io/fs"
	"testing"
	"testing/fstest"
)

func TestMountServesApplicationAndKitAssets(t *testing.T) {
	app := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("app")},
	}
	mounted := Mount(app)

	if got, err := fs.ReadFile(mounted, "index.html"); err != nil || string(got) != "app" {
		t.Fatalf("read app asset: got %q err=%v", got, err)
	}
	if got, err := fs.ReadFile(mounted, "desktopkit/tokens.css"); err != nil || len(got) == 0 {
		t.Fatalf("read kit asset: len=%d err=%v", len(got), err)
	}
}
