package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDoctorAndUpgradeKitToolPinsInScripts(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"go.mod":                        "module example.test/demo\n\ngo 1.26\n\nrequire github.com/wanstu/wails-desktop-kit v0.11.2\n",
		".github/workflows/release.yml": "jobs:\n  build:\n    uses: wanstu/wails-desktop-kit/.github/workflows/wails-desktop.yml@v0.11.2\n    with:\n      desktopkit-cli-version: v0.10.3\n",
		"scripts/package.sh":            "#!/bin/sh\ngo run github.com/wanstu/wails-desktop-kit/cmd/desktopkit@v0.10.2 package linux\n",
		"scripts/release.ps1":           "go install github.com/wanstu/wails-desktop-kit/cmd/desktopkit@v0.11.1\n",
		"Makefile":                      "build:\n\tgo run github.com/wanstu/wails-desktop-kit/cmd/desktopkit@v0.10.3 doctor\n",
		"scripts/dynamic.sh":            "go install github.com/wanstu/wails-desktop-kit/cmd/desktopkit@${KIT_VERSION}\n",
		"docs/guide.md":                 "go run github.com/wanstu/wails-desktop-kit/cmd/desktopkit@v0.10.1\n",
	}
	for name, body := range files {
		file := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
			t.Fatal(err)
		}
		perm := os.FileMode(0o644)
		if name == "scripts/package.sh" {
			perm = 0o755
		}
		if err := os.WriteFile(file, []byte(body), perm); err != nil {
			t.Fatal(err)
		}
	}
	inspection, err := inspectRepository(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := inspection.ScriptKitPins["scripts/package.sh"]; len(got) != 1 || got[0] != "v0.10.2" {
		t.Fatalf("missing script pin: %#v", inspection.ScriptKitPins)
	}
	if got := inspection.ScriptKitPins[".github/workflows/release.yml"]; len(got) != 1 || got[0] != "v0.10.3" {
		t.Fatalf("missing workflow helper pin: %#v", inspection.ScriptKitPins)
	}
	if _, found := inspection.ScriptKitPins["docs/guide.md"]; found {
		t.Fatal("must not treat documentation as build script")
	}
	if err := runDoctor([]string{"--root", root}); err == nil {
		t.Fatal("doctor must reject mixed script versions")
	}
	if err := runUpgrade([]string{"--root", root, "--to", "v0.11.3", "--tidy=false"}); err != nil {
		t.Fatal(err)
	}
	if err := runDoctor([]string{"--root", root}); err != nil {
		t.Fatalf("doctor after upgrade: %v", err)
	}
	for name, body := range files {
		actual, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(name, "docs/") || strings.HasSuffix(name, "dynamic.sh") {
			if string(actual) != body {
				t.Fatalf("%s changed unexpectedly: %s", name, actual)
			}
		} else if strings.Contains(string(actual), "v0.10.") || strings.Contains(string(actual), "@v0.11.1") || strings.Contains(string(actual), " v0.11.2") {
			t.Fatalf("%s has stale Kit version: %s", name, actual)
		}
	}
	if info, err := os.Stat(filepath.Join(root, "scripts", "package.sh")); err != nil || (runtime.GOOS != "windows" && info.Mode().Perm() != 0o755) {
		t.Fatalf("script executable bit lost: %v, %v", info, err)
	}
}

func TestScriptPinExtractionDoesNotModifyUnrelatedRefs(t *testing.T) {
	input := []byte("go install github.com/wanstu/wails-desktop-kit/cmd/desktopkit@v0.10.1\n" +
		"desktopkit-cli-version: 'v0.10.2' # text\n" +
		"other-tool@v0.10.1\n" +
		"go run github.com/wanstu/wails-desktop-kit/cmd/desktopkit@${KIT_VERSION}\n")
	pins := scriptKitPins(input)
	if len(pins) != 2 || pins[0] != "v0.10.1" || pins[1] != "v0.10.2" {
		t.Fatalf("pins=%v", pins)
	}
	got := string(rewriteScriptKitPins(input, "v0.11.3"))
	if strings.Count(got, "v0.11.3") != 2 || !strings.Contains(got, "other-tool@v0.10.1") || !strings.Contains(got, "@${KIT_VERSION}") {
		t.Fatalf("rewritten incorrectly: %s", got)
	}
}
