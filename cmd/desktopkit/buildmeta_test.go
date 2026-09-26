package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNumericProductVersion(t *testing.T) {
	cases := map[string]string{
		"v0.3.0-rc.1": "0.3.0",
		"1.2.3":       "1.2.3",
		"v10.20.30":   "10.20.30",
		"dev":         "0.0.0",
	}
	for input, want := range cases {
		if got := numericProductVersion(input); got != want {
			t.Fatalf("numericProductVersion(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestPrepareBuildMeta(t *testing.T) {
	root := t.TempDir()
	desktop := filepath.Join(root, "desktop")
	frontend := filepath.Join(desktop, "frontend")
	windows := filepath.Join(desktop, "build", "windows")
	for _, dir := range []string{frontend, windows} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(desktop, "wails.json"), []byte(`{"name":"demo","frontend:dir":"frontend"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(windows, "info.json"), []byte(`{"fixed":{"file_version":"{{.Info.ProductVersion}}"},"info":{"0000":{"ProductVersion":"{{.Info.ProductVersion}}","ProductName":"{{.Info.ProductName}}"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := prepareBuildMeta(root, "desktop", "v0.3.0-rc.1", "1234567890abcdef"); err != nil {
		t.Fatal(err)
	}

	var wails map[string]any
	readJSONFile(t, filepath.Join(desktop, "wails.json"), &wails)
	info := wails["info"].(map[string]any)
	if info["productVersion"] != "0.3.0" {
		t.Fatalf("wails productVersion = %#v", info["productVersion"])
	}

	var meta buildMetadata
	readJSONFile(t, filepath.Join(frontend, "desktopkit-build-info.json"), &meta)
	if meta.Version != "v0.3.0-rc.1" || meta.ProductVersion != "0.3.0" || meta.Commit != "1234567890abcdef" {
		t.Fatalf("metadata = %#v", meta)
	}

	var win map[string]any
	readJSONFile(t, filepath.Join(windows, "info.json"), &win)
	fixed := win["fixed"].(map[string]any)
	if fixed["file_version"] != "0.3.0" {
		t.Fatalf("file_version = %#v", fixed["file_version"])
	}
	lang := win["info"].(map[string]any)["0000"].(map[string]any)
	if lang["ProductVersion"] != "0.3.0-rc.1" {
		t.Fatalf("ProductVersion = %#v", lang["ProductVersion"])
	}
}

func readJSONFile(t *testing.T, path string, target any) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatal(err)
	}
}
