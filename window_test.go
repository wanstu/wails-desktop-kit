package desktopkit

import "testing"

func TestDefaultWindowConfigRetainsTrayOnLinux(t *testing.T) {
	cfg := DefaultWindowConfig()
	if cfg.HidePolicy != HideAlways {
		t.Fatalf("default hide policy = %q, want %q", cfg.HidePolicy, HideAlways)
	}
	if !CanHideToTray(cfg.HidePolicy, true, "linux") {
		t.Fatal("default window config should allow hide-to-tray on Linux when tray is enabled")
	}
}

func TestCanHideToTraySafePolicy(t *testing.T) {
	tests := []struct {
		goos string
		want bool
	}{
		{"windows", true},
		{"darwin", true},
		{"linux", false},
		{"freebsd", false},
	}
	for _, tt := range tests {
		if got := CanHideToTray(HideSafe, true, tt.goos); got != tt.want {
			t.Fatalf("CanHideToTray(HideSafe, true, %q) = %v, want %v", tt.goos, got, tt.want)
		}
	}
}

func TestCanHideToTrayRequiresTray(t *testing.T) {
	if CanHideToTray(HideAlways, false, "windows") {
		t.Fatal("hide should be disabled without tray")
	}
}
