package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunIconPrepareWails(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source.png")
	if err := os.WriteFile(source, []byte("png-placeholder"), 0o644); err != nil {
		t.Fatal(err)
	}
	windowsDir := filepath.Join(root, "build", "windows")
	if err := os.MkdirAll(windowsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ico := filepath.Join(windowsDir, "icon.ico")
	if err := os.WriteFile(ico, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runIconPrepareWails([]string{"--input", source, "--desktop-dir", root}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "build", "appicon.png")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ico); !os.IsNotExist(err) {
		t.Fatalf("generated ico still exists: %v", err)
	}
}

func TestParseExtraBinaries(t *testing.T) {
	got, err := parseExtraBinaries([]string{"helper=build/helper", "cli=dist/cli"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].InstallName != "helper" {
		t.Fatalf("got %#v", got)
	}
	if _, err := parseExtraBinaries([]string{"bad"}); err == nil {
		t.Fatal("expected malformed extra bin error")
	}
}
