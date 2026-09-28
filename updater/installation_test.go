package updater

import "testing"

func TestInstallationManaged(t *testing.T) {
	if (Installation{Mode: InstallationPortable}).Managed() {
		t.Fatal("portable installation must not be managed")
	}
	if !(Installation{Mode: InstallationUser}).Managed() {
		t.Fatal("user installation should be managed")
	}
	if !(Installation{Mode: InstallationMachine}).Managed() {
		t.Fatal("machine installation should be managed")
	}
}

func TestCurrentInstallationRejectsUnsafeAppID(t *testing.T) {
	for _, appID := range []string{"", "  ", "demo/app", `demo\app`} {
		if _, err := CurrentInstallation(appID); err == nil {
			t.Fatalf("expected invalid app id error for %q", appID)
		}
	}
}
