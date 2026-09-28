//go:build windows

package updater

import "testing"

func TestCleanWindowsRegistryPath(t *testing.T) {
	tests := map[string]string{
		`"C:\Apps\Demo\demo.exe"`:   `C:\Apps\Demo\demo.exe`,
		`"C:\Apps\Demo\demo.exe",0`: `C:\Apps\Demo\demo.exe`,
		`C:\Apps\Demo\demo.exe`:     `C:\Apps\Demo\demo.exe`,
	}
	for input, want := range tests {
		if got := cleanWindowsRegistryPath(input); got != want {
			t.Fatalf("cleanWindowsRegistryPath(%q) = %q want %q", input, got, want)
		}
	}
}

func TestSameWindowsPathIsCaseInsensitive(t *testing.T) {
	if !sameWindowsPath(`C:\Users\Demo\App.exe`, `c:\users\demo\APP.EXE`) {
		t.Fatal("expected Windows paths to match case-insensitively")
	}
}
