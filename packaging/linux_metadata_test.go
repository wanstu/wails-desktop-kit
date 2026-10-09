package packaging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxDebMetadataAndCopyright(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "demo")
	if err := os.WriteFile(binary, make([]byte, 5200), 0755); err != nil {
		t.Fatal(err)
	}
	notice := filepath.Join(dir, "COPYRIGHT")
	if err := os.WriteFile(notice, []byte("Copyright 2026 Example\nLicense: LicenseRef-Proprietary\n"), 0644); err != nil {
		t.Fatal(err)
	}
	appstream := filepath.Join(dir, "demo.metainfo.xml")
	const metadata = `<?xml version="1.0"?><component type="console-application"><id>demo</id><project_license>MIT</project_license><metadata_license>CC0-1.0</metadata_license></component>`
	if err := os.WriteFile(appstream, []byte(metadata), 0644); err != nil {
		t.Fatal(err)
	}
	arts, err := PackageLinux(LinuxRequest{Input: binary, OutputDir: dir, AppName: "demo", AssetBase: "demo",
		PackageName: "demo", Architecture: "amd64", CopyrightFile: notice, AppStreamFile: appstream, Formats: []Format{FormatDeb}})
	if err != nil {
		t.Fatal(err)
	}
	ar := readArFile(t, arts[0].Path)
	control := readTarGzBytes(t, ar["control.tar.gz"])
	if !strings.Contains(string(control["./control"]), "Installed-Size: ") {
		t.Fatalf("missing actual installed size: %s", control["./control"])
	}
	data := readTarGzBytes(t, ar["data.tar.gz"])
	if string(data["./usr/share/doc/demo/copyright"]) != "Copyright 2026 Example\nLicense: LicenseRef-Proprietary\n" {
		t.Fatal("copyright document was not packaged")
	}
	if string(data["./usr/share/metainfo/demo.metainfo.xml"]) != metadata {
		t.Fatal("AppStream metadata was not packaged")
	}
}

func TestLinuxServiceEnvironmentOverrides(t *testing.T) {
	req := LinuxRequest{AppName: "demo", PackageName: "demo", Description: "demo",
		Systemd: &LinuxSystemdService{
			Args:            []string{"--listen", "${DEMO_LISTEN}", "--data-dir", "/var/lib/demo"},
			Environment:     []string{"DEMO_LISTEN=0.0.0.0:8610"},
			EnvironmentFile: "/etc/default/demo",
		}}
	text := string(linuxSystemdUnit(req))
	for _, want := range []string{"Environment=\"DEMO_LISTEN=0.0.0.0:8610\"", "EnvironmentFile=-/etc/default/demo", "\"--listen\" \"${DEMO_LISTEN}\""} {
		if !strings.Contains(text, want) {
			t.Fatalf("systemd unit missing %q:\n%s", want, text)
		}
	}
	req.Systemd.EnvironmentFile = "/etc/shadow"
	if err := validateLinuxSystemdService(LinuxRequest{Input: "example", AppName: "demo", PackageName: "demo", Formats: []Format{FormatDeb}, Systemd: req.Systemd}); err == nil {
		t.Fatal("unsafe environment file allowed")
	}
}
