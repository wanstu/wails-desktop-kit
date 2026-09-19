package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndCopy(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "settings.json")
	if err := Write(path, []byte("one\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Write(path, []byte("two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "two\n" {
		t.Fatalf("got %q", data)
	}
	copyPath := filepath.Join(dir, "copy.json")
	if err := Copy(path, copyPath, 0o600); err != nil {
		t.Fatal(err)
	}
	copied, err := os.ReadFile(copyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(copied) != string(data) {
		t.Fatalf("copy got %q", copied)
	}
}
