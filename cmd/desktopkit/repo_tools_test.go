package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectRepositoryAndUpgrade(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".github", "workflows"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module demo\n\ngo 1.26\n\nrequire (\n\tgithub.com/wailsapp/wails/v2 v2.15.0\n\tgithub.com/wanstu/wails-desktop-kit v0.7.2\n)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	workflow := "jobs:\n  desktop:\n    uses: wanstu/wails-desktop-kit/.github/workflows/wails-desktop.yml@v0.7.2\n    with:\n      app-name: demo\n"
	if err := os.WriteFile(filepath.Join(root, ".github", "workflows", "ci.yml"), []byte(workflow), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "wails.json"), []byte(`{"outputfilename":"demo"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	inspection, err := inspectRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.KitModuleVersion != "v0.7.2" || inspection.WailsVersion != "v2.15.0" {
		t.Fatalf("inspection = %#v", inspection)
	}
	if err := runUpgrade([]string{"--root", root, "--to", "v0.8.0", "--tidy=false"}); err != nil {
		t.Fatal(err)
	}
	inspection, err = inspectRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.KitModuleVersion != "v0.8.0" || inspection.WorkflowVersions[".github/workflows/ci.yml"] != "v0.8.0" {
		t.Fatalf("upgraded inspection = %#v", inspection)
	}
}
