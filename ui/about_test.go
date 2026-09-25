package ui

import (
	"io/fs"
	"strings"
	"testing"
)

func TestAboutComponentLoadsBuildMetadata(t *testing.T) {
	data, err := fs.ReadFile(embedded, "assets/about.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(data)
	for _, want := range []string{
		"desktopkit-build-info.json",
		"customElements",
		"dk-about",
		"Product ",
		"Commit ",
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("about.js missing %q", want)
		}
	}
}
