package theme

import "testing"

func TestPreference(t *testing.T) {
	got := NormalizePreference(Preference{Mode: "DARK", Pack: " Ocean "})
	if got.Mode != ModeDark || got.Pack != "ocean" {
		t.Fatalf("NormalizePreference() = %#v", got)
	}
	if err := ValidatePreference(got); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePackName("../bad"); err == nil {
		t.Fatal("expected invalid pack")
	}
}
