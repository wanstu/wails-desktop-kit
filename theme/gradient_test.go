package theme

import (
	"strings"
	"testing"
)

func TestFallbackManifestHasUsableGradientColors(t *testing.T) {
	manifest := fallbackManifest()
	if len(manifest.Packs) != 4 {
		t.Fatalf("fallback themes = %d", len(manifest.Packs))
	}
	for _, pack := range manifest.Packs {
		if !gradientHexColorPattern.MatchString(pack.GradientLight) ||
			!gradientHexColorPattern.MatchString(pack.GradientDark) {
			t.Fatalf("%s missing gradient palette: %+v", pack.Name, pack)
		}
	}
}

func TestValidateManifestAcceptsLegacyAndRejectsInvalidGradientColors(t *testing.T) {
	manifest := fallbackManifest()
	manifest.BaseURL = "https://example.com/themes/"
	manifest.Packs[0].GradientLight = ""
	manifest.Packs[0].GradientDark = ""
	if err := validateManifest(manifest); err != nil {
		t.Fatalf("legacy pack rejected: %v", err)
	}
	manifest.Packs[0].GradientLight = "#abcdef"
	if err := validateManifest(manifest); err == nil || !strings.Contains(err.Error(), "both") {
		t.Fatalf("missing pair should fail: %v", err)
	}
	manifest.Packs[0].GradientDark = "#GGGGGG"
	if err := validateManifest(manifest); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("invalid gradient color should fail: %v", err)
	}
}
