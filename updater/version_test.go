package updater

import "testing"

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  int
	}{
		{"v1.2.3", "1.2.2", 1},
		{"1.2.3", "v1.2.3", 0},
		{"1.2.3-rc.1", "1.2.3", -1},
		{"1.2.3", "1.2.3-rc.9", 1},
		{"1.2.3-rc.10", "1.2.3-rc.2", 1},
		{"1.2.3-alpha.1", "1.2.3-alpha.beta", -1},
		{"1.2.3+build.5", "1.2.3+build.9", 0},
		{"2.0.0-rc.1", "1.99.99", 1},
	}
	for _, test := range tests {
		got, err := CompareVersions(test.left, test.right)
		if err != nil {
			t.Fatalf("CompareVersions(%q, %q): %v", test.left, test.right, err)
		}
		if got != test.want {
			t.Fatalf("CompareVersions(%q, %q) = %d want %d", test.left, test.right, got, test.want)
		}
	}
}

func TestCompareVersionsRejectsInvalidVersion(t *testing.T) {
	if _, err := CompareVersions("dev", "1.0.0"); err == nil {
		t.Fatal("expected invalid version error")
	}
	if _, err := CompareVersions("1.0", "1.0.0"); err == nil {
		t.Fatal("expected incomplete semver error")
	}
}
