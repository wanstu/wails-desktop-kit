package desktopkit

import "testing"

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
