package packaging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLinuxSystemdDebIncludesServiceAndLifecycle(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "server")
	if err := os.WriteFile(binary, []byte("test-linux-exe"), 0755); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	req := LinuxRequest{
		Input: binary, OutputDir: out, AppName: "mcp-center", AssetBase: "mcp-center-v0.1.1",
		PackageName: "mcp-center", PackageVersion: "0.1.1", Architecture: "amd64",
		Description: "MCP Center service", Maintainer: "Kit Test",
		Formats: []Format{FormatDeb},
		Systemd: &LinuxSystemdService{Args: []string{"--listen", "0.0.0.0:8610", "--data-dir", "/var/lib/mcp-center"}},
	}
	artifacts, err := PackageLinux(req)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 1 {
		t.Fatal(artifacts)
	}
	assertChecksum(t, artifacts[0].Path)
	ar := readArFile(t, artifacts[0].Path)
	ctr := readTarGzBytes(t, ar["control.tar.gz"])
	data := readTarGzBytes(t, ar["data.tar.gz"])
	if !strings.Contains(string(ctr["./control"]), "Version: 0.1.1") {
		t.Fatal("missing version")
	}
	for _, file := range []string{"./postinst", "./prerm", "./postrm"} {
		if len(ctr[file]) == 0 {
			t.Fatalf("missing lifecycle script %s", file)
		}
		if !strings.Contains(string(ctr[file]), "#!/bin/sh") {
			t.Fatal(file)
		}
	}
	if !strings.Contains(string(ctr["./postinst"]), "systemctl enable 'mcp-center.service'") ||
		!strings.Contains(string(ctr["./postinst"]), "systemctl start 'mcp-center.service'") ||
		!strings.Contains(string(ctr["./postinst"]), "systemctl try-restart 'mcp-center.service'") ||
		!strings.Contains(string(ctr["./postinst"]), "install -d -m 0700") {
		t.Fatal("install lifecycle missing")
	}
	if !strings.Contains(string(ctr["./prerm"]), "disable --now") {
		t.Fatal("uninstall lifecycle missing")
	}
	if strings.Contains(string(ctr["./postrm"]), "rm -rf") {
		t.Fatal("data deletion forbidden")
	}
	unit := string(data["./lib/systemd/system/mcp-center.service"])
	for _, want := range []string{
		"ExecStart=\"/usr/bin/mcp-center\" \"--listen\" \"0.0.0.0:8610\" \"--data-dir\" \"/var/lib/mcp-center\"",
		"User=mcp-center", "Group=mcp-center", "WorkingDirectory=/var/lib/mcp-center",
		"Restart=on-failure", "WantedBy=multi-user.target", "NoNewPrivileges=true", "UMask=0077",
	} {
		if !strings.Contains(unit, want) {
			t.Fatalf("missing %q in unit:\n%s", want, unit)
		}
	}
	if string(data["./usr/bin/mcp-center"]) != "test-linux-exe" {
		t.Fatal("wrong binary")
	}
}

func TestLinuxSystemdServiceIsOptIn(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app")
	if err := os.WriteFile(file, []byte("app"), 0755); err != nil {
		t.Fatal(err)
	}
	arts, err := PackageLinux(LinuxRequest{Input: file, OutputDir: dir, AppName: "app", AssetBase: "app", Formats: []Format{FormatDeb}})
	if err != nil {
		t.Fatal(err)
	}
	ar := readArFile(t, arts[0].Path)
	ctr := readTarGzBytes(t, ar["control.tar.gz"])
	data := readTarGzBytes(t, ar["data.tar.gz"])
	if len(ctr) != 1 || data["./lib/systemd/system/app.service"] != nil {
		t.Fatal("desktop package unexpectedly got systemd scripts")
	}
}

func TestLinuxSystemdRejectsUnsafeInput(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "app")
	if err := os.WriteFile(file, []byte("app"), 0755); err != nil {
		t.Fatal(err)
	}
	base := LinuxRequest{Input: file, OutputDir: dir, AppName: "app", AssetBase: "app", Formats: []Format{FormatDeb}}
	for _, test := range []LinuxSystemdService{
		{Name: "../../root"},
		{User: "root;rm"},
		{Group: "hack name"},
		{DataDir: "/etc"},
		{DataDir: "/var/lib/../etc"},
		{Args: []string{"--data-dir\nExecStart=/bin/sh"}},
		{Description: "test\nExecStart=/bin/sh"},
	} {
		base.Systemd = &test
		if _, err := PackageLinux(base); err == nil {
			t.Fatalf("accepted unsafe config %#v", test)
		}
	}
}

func TestSystemdArgumentEscapesSpecifiersAndQuotes(t *testing.T) {
	got := systemdExecArgument("with %i and \"quotes\"")
	if got != "\"with %%i and \\\"quotes\\\"\"" {
		t.Fatalf("bad escaping: %q", got)
	}
}
