package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunIconGenerate(t *testing.T) {
	output := filepath.Join(t.TempDir(), "appicon.png")
	if err := run([]string{
		"icon", "generate",
		"--output", output,
		"--symbol", "terminal",
		"--background", "#2463EB",
	}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("generated icon is empty")
	}
}

func TestRunIconGenerateMonogram(t *testing.T) {
	output := filepath.Join(t.TempDir(), "appicon.png")
	if err := run([]string{
		"icon", "generate",
		"--output", output,
		"--symbol", "monogram",
		"--text", "F",
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRunIconGenerateRequiresOutput(t *testing.T) {
	if err := run([]string{"icon", "generate"}); err == nil {
		t.Fatal("expected missing output error")
	}
}

func TestRunIconLegacyNormalizeSyntaxStillWorks(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "source.png")
	output := filepath.Join(dir, "normalized.png")

	if err := run([]string{
		"icon", "generate",
		"--output", input,
		"--symbol", "terminal",
	}); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{
		"icon",
		"--input", input,
		"--output", output,
		"--canvas", "64",
		"--fill", "0.8",
	}); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(output); err != nil || info.Size() == 0 {
		t.Fatalf("legacy normalize output: info=%v err=%v", info, err)
	}
}
