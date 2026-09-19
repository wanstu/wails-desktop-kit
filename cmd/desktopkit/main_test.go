package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunHelp(t *testing.T) {
	if err := run([]string{"--help"}); err != nil {
		t.Fatal(err)
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	if err := run([]string{"nope"}); err == nil {
		t.Fatal("expected unknown command error")
	}
}

func TestRunPackageLinux(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "demo")
	if err := os.WriteFile(input, []byte("#!/bin/sh\necho demo\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	dist := filepath.Join(root, "dist")
	if err := run([]string{
		"package", "linux",
		"--input", input,
		"--dist", dist,
		"--app-name", "demo",
		"--asset-base", "demo-v1.0.0",
		"--package-version", "1.0.0",
		"--formats", "raw,deb,tar.gz",
		"--description", "Demo",
		"--maintainer", "Desktop Kit Test",
	}); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"demo-v1.0.0-linux-amd64",
		"demo-v1.0.0-linux-amd64.deb",
		"demo-v1.0.0-linux-amd64.tar.gz",
	} {
		path := filepath.Join(dist, name)
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Fatalf("artifact %s: info=%v err=%v", name, info, err)
		}
		if _, err := os.Stat(path + ".sha256"); err != nil {
			t.Fatalf("checksum for %s: %v", name, err)
		}
	}
}
