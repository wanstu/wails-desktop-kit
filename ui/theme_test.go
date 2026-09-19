package ui

import (
	"io/fs"
	"strings"
	"testing"
)

func TestThemeTokensExposeDarkThemeContract(t *testing.T) {
	data, err := fs.ReadFile(embedded, "assets/tokens.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(data)
	for _, want := range []string{
		`:root[data-dk-theme="dark"]`,
		"--dk-nav-active-bg:",
		"--dk-danger-border:",
		"--dk-control-track:",
		"--dk-backdrop:",
		"--dk-log-bg:",
	} {
		if !strings.Contains(css, want) {
			t.Fatalf("tokens.css missing %q", want)
		}
	}
}

func TestThemeScriptSupportsExplicitModesWithoutPersistence(t *testing.T) {
	data, err := fs.ReadFile(embedded, "assets/theme.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(data)
	for _, want := range []string{
		`desktopKitTheme`,
		`light`,
		`dark`,
		`system`,
		`matchMedia`,
		`prefers-color-scheme: dark`,
		`data-dk-theme`,
		`data-dk-theme-pack`,
		`setPack`,
		`getPack`,
		`clearPack`,
		`applyPack`,
		`loadCatalog`,
		`refreshCatalog`,
		`getCatalog`,
	} {
		if !strings.Contains(js, want) {
			t.Fatalf("theme.js missing %q", want)
		}
	}
	if strings.Contains(js, "localStorage") || strings.Contains(js, "sessionStorage") {
		t.Fatal("theme.js must not own application theme persistence")
	}
	if strings.Contains(js, "midnight") || strings.Contains(js, "graphite") || strings.Contains(js, "forest") {
		t.Fatal("theme.js must define the pack protocol, not product or optional pack names")
	}
}

func TestSharedComponentsUseThemeTokens(t *testing.T) {
	for _, name := range []string{"assets/components.css", "assets/navigation.css"} {
		data, err := fs.ReadFile(embedded, name)
		if err != nil {
			t.Fatal(err)
		}
		css := string(data)
		for _, forbidden := range []string{
			"#f3c4c0",
			"#f5c9c5",
			"#8bb0f8",
			"#cbd5e1",
			"#4b5563",
			"#d5dbe4",
			"#f3f4f6",
		} {
			if strings.Contains(strings.ToLower(css), strings.ToLower(forbidden)) {
				t.Fatalf("%s contains theme-specific fixed color %s", name, forbidden)
			}
		}
	}
}
