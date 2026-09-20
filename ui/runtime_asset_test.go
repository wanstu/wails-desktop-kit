package ui

import (
	"io/fs"
	"strings"
	"testing"
)

func TestRuntimeClipboardPrefersNativeWailsClipboard(t *testing.T) {
	source, err := fs.ReadFile(embedded, "assets/runtime.js")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)

	native := strings.Index(text, "runtime.ClipboardSetText")
	browser := strings.Index(text, "navigator.clipboard.writeText")
	fallback := strings.Index(text, "document.execCommand")
	if native < 0 || browser < 0 || fallback < 0 {
		t.Fatalf("runtime clipboard fallbacks missing: native=%d browser=%d fallback=%d", native, browser, fallback)
	}
	if !(native < browser && browser < fallback) {
		t.Fatalf("clipboard fallback order must be native -> browser -> execCommand: native=%d browser=%d fallback=%d", native, browser, fallback)
	}
	for _, required := range []string{
		"runtime.ClipboardGetText",
		"navigator.clipboard.readText",
		"kit.clipboard = Object.freeze",
	} {
		if !strings.Contains(text, required) {
			t.Fatalf("runtime clipboard asset missing %q", required)
		}
	}
}
