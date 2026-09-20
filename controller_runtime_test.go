package desktopkit

import (
	"errors"
	"testing"
)

func TestClipboardRuntimeRequiresReadyContext(t *testing.T) {
	controller := newController(false, false)

	if err := controller.ClipboardSetText("test"); !errors.Is(err, ErrRuntimeNotReady) {
		t.Fatalf("ClipboardSetText error = %v, want ErrRuntimeNotReady", err)
	}
	if text, err := controller.ClipboardGetText(); text != "" || !errors.Is(err, ErrRuntimeNotReady) {
		t.Fatalf("ClipboardGetText = %q, %v; want empty, ErrRuntimeNotReady", text, err)
	}
}
