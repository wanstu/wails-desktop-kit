package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateFilesIfMissing(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "old")
	target := filepath.Join(root, "new")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "settings.json"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	count, err := MigrateFilesIfMissing(source, target, "settings.json", "missing.json")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count = %d", count)
	}
	data, err := os.ReadFile(filepath.Join(target, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Fatalf("data = %q", data)
	}
	if err := os.WriteFile(filepath.Join(source, "settings.json"), []byte("newer"), 0o600); err != nil {
		t.Fatal(err)
	}
	count, err = MigrateFilesIfMissing(source, target, "settings.json")
	if err != nil || count != 0 {
		t.Fatalf("second migration = %d, %v", count, err)
	}
}

func TestMigrateTreeMissing(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "old")
	target := filepath.Join(root, "new")
	if err := os.MkdirAll(filepath.Join(source, "instances", "a"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "instances", "a", "state.json"), []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "instances", "a"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "instances", "a", "keep.json"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	count, err := MigrateTreeMissing(filepath.Join(source, "instances"), filepath.Join(target, "instances"))
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count = %d", count)
	}
	if _, err := os.Stat(filepath.Join(target, "instances", "a", "state.json")); err != nil {
		t.Fatal(err)
	}
}
