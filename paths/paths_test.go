package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateAppID(t *testing.T) {
	for _, id := range []string{"ssh-client", "frp_client", "adm.v2", "App1"} {
		if err := ValidateAppID(id); err != nil {
			t.Fatalf("%q rejected: %v", id, err)
		}
	}
	for _, id := range []string{"", ".hidden", "../evil", "a/b", "a\\b", "has space", "应用"} {
		if err := ValidateAppID(id); err == nil {
			t.Fatalf("%q should be rejected", id)
		}
	}
}

func TestConfigRootUsesXDGConfigHome(t *testing.T) {
	root := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", root)
	got, err := ConfigRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(root) {
		t.Fatalf("got %q want %q", got, filepath.Clean(root))
	}
}

func TestConfigRootRejectsRelativeXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "relative/path")
	if _, err := ConfigRoot(); err == nil {
		t.Fatal("expected relative XDG_CONFIG_HOME error")
	}
}

func TestConfigDirFallsBackToDotConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	var homeKey string
	if runtime.GOOS == "windows" {
		homeKey = "USERPROFILE"
	} else {
		homeKey = "HOME"
	}
	home := t.TempDir()
	t.Setenv(homeKey, home)

	got, err := ConfigDir("ssh-client")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".config", "ssh-client")
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEnsureConfigDir(t *testing.T) {
	root := filepath.Join(t.TempDir(), "xdg")
	t.Setenv("XDG_CONFIG_HOME", root)
	dir, err := EnsureConfigDir("ssh-client")
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", dir)
	}
}
