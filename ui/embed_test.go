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
	for _, name := range []string{"desktopkit/tokens.css", "desktopkit/theme.js"} {
		if got, err := fs.ReadFile(mounted, name); err != nil || len(got) == 0 {
			t.Fatalf("read kit asset %s: len=%d err=%v", name, len(got), err)
		}
	}
}
