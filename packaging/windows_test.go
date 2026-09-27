package packaging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageWindowsBuildsSetupFromExistingExecutable(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "demo.exe")
	if err := os.WriteFile(input, []byte("already-built-product-exe"), 0o644); err != nil {
		t.Fatal(err)
	}
	dist := filepath.Join(root, "dist")
	expected := filepath.Join(dist, "demo-v1.2.3-windows-amd64-setup.exe")

	originalCompiler := compileWindowsInstaller
	defer func() { compileWindowsInstaller = originalCompiler }()
	compileWindowsInstaller = func(executable, scriptPath string) error {
		if executable != "fake-makensis" {
			t.Fatalf("compiler = %q", executable)
		}
		script, err := os.ReadFile(scriptPath)
		if err != nil {
			return err
		}
		text := string(script)
		for _, want := range []string{
			"RequestExecutionLevel user",
			"InstallDir \"$LOCALAPPDATA\\Programs\\${PRODUCT_NAME}\"",
			"File /oname=\"${APP_EXE}\"",
			"WriteRegStr HKCU",
			"CreateShortcut \"$SMPROGRAMS\\${PRODUCT_NAME}\\${PRODUCT_NAME}.lnk\"",
		} {
			if !strings.Contains(text, want) {
				t.Fatalf("installer script missing %q:\n%s", want, text)
			}
		}
		if strings.Contains(text, "RMDir /r") {
			t.Fatalf("installer must not recursively delete application/user data:\n%s", text)
		}
		return os.WriteFile(expected, []byte("compiled-installer"), 0o644)
	}

	artifact, err := PackageWindows(WindowsRequest{
		Input:             input,
		OutputDir:         dist,
		AppName:           "demo",
		AssetBase:         "demo-v1.2.3",
		PackageVersion:    "v1.2.3-rc.1",
		Architecture:      "amd64",
		InstallScope:      "user",
		ProductName:       "Demo App",
		Publisher:         "Desktop Kit",
		AppID:             "demo",
		StartMenuShortcut: true,
		NSISPath:          "fake-makensis",
	})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Path != expected {
		t.Fatalf("artifact path = %q want %q", artifact.Path, expected)
	}
	if artifact.Format != FormatWindowsSetup {
		t.Fatalf("artifact format = %q", artifact.Format)
	}
	assertChecksum(t, expected)
}

func TestWindowsInstallerMachineScopeAndOptionalDesktopShortcut(t *testing.T) {
	root := t.TempDir()
	scriptPath := filepath.Join(root, "installer.nsi")
	err := writeWindowsInstallerScript(scriptPath, windowsTemplateData{
		Input:             "C:\\build\\demo.exe",
		Output:            "C:\\dist\\demo-setup.exe",
		AppExe:            "demo.exe",
		AppID:             "demo",
		ProductName:       "Demo App",
		Publisher:         "Desktop Kit",
		DisplayVersion:    "1.2.3",
		NumericVersion:    "1.2.3.0",
		InstallScope:      "machine",
		StartMenuShortcut: true,
		DesktopShortcut:   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"RequestExecutionLevel admin",
		"InstallDir \"$PROGRAMFILES64\\${PUBLISHER}\\${PRODUCT_NAME}\"",
		"WriteRegStr HKLM",
		"CreateShortcut \"$DESKTOP\\${PRODUCT_NAME}.lnk\"",
		"SetRegView 64",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("installer script missing %q:\n%s", want, text)
		}
	}
}

func TestPackageWindowsRejectsInvalidScope(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "demo.exe")
	if err := os.WriteFile(input, []byte("demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := PackageWindows(WindowsRequest{
		Input:        input,
		OutputDir:    filepath.Join(root, "dist"),
		AppName:      "demo",
		AssetBase:    "demo",
		InstallScope: "system",
		NSISPath:     "fake",
	})
	if err == nil || !strings.Contains(err.Error(), "install scope") {
		t.Fatalf("expected install scope error, got %v", err)
	}
}

func TestNumericWindowsVersion(t *testing.T) {
	for input, want := range map[string]string{
		"v1.2.3":      "1.2.3.0",
		"1.2.3-rc.1":  "1.2.3.0",
		"v10.20.30+x": "10.20.30.0",
		"dev":         "0.0.0.0",
	} {
		if got := numericWindowsVersion(input); got != want {
			t.Fatalf("numericWindowsVersion(%q) = %q want %q", input, got, want)
		}
	}
}
