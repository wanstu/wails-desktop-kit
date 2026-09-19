package packaging

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestParseFormats(t *testing.T) {
	formats, err := ParseFormats("raw, deb,tar.gz,deb")
	if err != nil {
		t.Fatal(err)
	}
	want := []Format{FormatRaw, FormatDeb, FormatTarGz}
	if fmt.Sprint(formats) != fmt.Sprint(want) {
		t.Fatalf("formats = %v want %v", formats, want)
	}
	if _, err := ParseFormats("appimage"); err == nil {
		t.Fatal("expected unsupported format error")
	}
}

func TestPackageLinuxProducesRawTarAndDeb(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "demo")
	if err := os.WriteFile(input, []byte("#!/bin/sh\necho demo\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	desktop := filepath.Join(root, "demo.desktop")
	if err := os.WriteFile(desktop, []byte("[Desktop Entry]\nType=Application\nName=Demo\nExec=demo\nIcon=demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	icon := filepath.Join(root, "demo.png")
	if err := os.WriteFile(icon, []byte("fake-png-for-packaging-test"), 0o644); err != nil {
		t.Fatal(err)
	}

	dist := filepath.Join(root, "dist")
	artifacts, err := PackageLinux(LinuxRequest{
		Input:          input,
		OutputDir:      dist,
		AppName:        "demo",
		AssetBase:      "demo-v1.2.3",
		PackageName:    "demo",
		PackageVersion: "1.2.3",
		Architecture:   "amd64",
		Description:    "Demo app\nSecond line",
		Maintainer:     "Demo Maintainer",
		Depends:        "libgtk-3-0",
		DesktopFile:    desktop,
		IconFile:       icon,
		Formats:        []Format{FormatRaw, FormatTarGz, FormatDeb},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 3 {
		t.Fatalf("artifact count = %d", len(artifacts))
	}
	for _, artifact := range artifacts {
		if artifact.Size <= 0 {
			t.Fatalf("empty artifact: %#v", artifact)
		}
		assertChecksum(t, artifact.Path)
	}

	raw := filepath.Join(dist, "demo-v1.2.3-linux-amd64")
	info, err := os.Stat(raw)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("raw binary is not executable: %v", info.Mode())
	}

	tarEntries := readTarGzFile(t, filepath.Join(dist, "demo-v1.2.3-linux-amd64.tar.gz"))
	for _, name := range []string{
		"demo-v1.2.3-linux-amd64/demo",
		"demo-v1.2.3-linux-amd64/demo.desktop",
		"demo-v1.2.3-linux-amd64/demo.png",
	} {
		if _, ok := tarEntries[name]; !ok {
			t.Fatalf("tar.gz missing %s; entries=%v", name, keys(tarEntries))
		}
	}

	debEntries := readArFile(t, filepath.Join(dist, "demo-v1.2.3-linux-amd64.deb"))
	if string(debEntries["debian-binary"]) != "2.0\n" {
		t.Fatalf("invalid debian-binary: %q", debEntries["debian-binary"])
	}
	control := readTarGzBytes(t, debEntries["control.tar.gz"])
	controlText := string(control["./control"])
	if !strings.HasSuffix(controlText, "\n") {
		t.Fatalf("control must end with a real newline: %q", controlText)
	}
	if strings.Contains(controlText, `Description: Demo app\n`) {
		t.Fatalf("control contains a literal escaped newline: %q", controlText)
	}
	for _, want := range []string{
		"Package: demo",
		"Version: 1.2.3",
		"Architecture: amd64",
		"Depends: libgtk-3-0",
		"Description: Demo app",
		" Second line",
	} {
		if !strings.Contains(controlText, want) {
			t.Fatalf("control missing %q:\n%s", want, controlText)
		}
	}
	data := readTarGzBytes(t, debEntries["data.tar.gz"])
	for _, name := range []string{
		"./usr/bin/demo",
		"./usr/share/applications/demo.desktop",
		"./usr/share/icons/hicolor/256x256/apps/demo.png",
	} {
		if _, ok := data[name]; !ok {
			t.Fatalf("deb data missing %s; entries=%v", name, keys(data))
		}
	}
}

func TestPackageLinuxRejectsUnsafeMetadata(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "demo")
	if err := os.WriteFile(input, []byte("demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := PackageLinux(LinuxRequest{
		Input:          input,
		OutputDir:      root,
		AppName:        "demo",
		AssetBase:      "demo",
		PackageName:    "../bad",
		PackageVersion: "1.0.0",
		Formats:        []Format{FormatDeb},
	})
	if err == nil {
		t.Fatal("expected invalid package name error")
	}
}

func assertChecksum(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	checksum, err := os.ReadFile(path + ".sha256")
	if err != nil {
		t.Fatal(err)
	}
	wantPrefix := hex.EncodeToString(sum[:]) + "  " + filepath.Base(path)
	if strings.TrimSpace(string(checksum)) != wantPrefix {
		t.Fatalf("checksum = %q want %q", checksum, wantPrefix)
	}
}

func readTarGzFile(t *testing.T, path string) map[string][]byte {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	return readTarGz(t, file)
}

func readTarGzBytes(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	return readTarGz(t, bytes.NewReader(data))
}

func readTarGz(t *testing.T, reader io.Reader) map[string][]byte {
	t.Helper()
	gz, err := gzip.NewReader(reader)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	out := map[string][]byte{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		out[header.Name] = data
	}
	return out
}

func readArFile(t *testing.T, path string) map[string][]byte {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	magic := make([]byte, 8)
	if _, err := io.ReadFull(reader, magic); err != nil {
		t.Fatal(err)
	}
	if string(magic) != "!<arch>\n" {
		t.Fatalf("bad ar magic: %q", magic)
	}
	out := map[string][]byte{}
	for {
		header := make([]byte, 60)
		_, err := io.ReadFull(reader, header)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if string(header[58:60]) != "`\n" {
			t.Fatalf("bad ar header trailer")
		}
		name := strings.TrimSuffix(strings.TrimSpace(string(header[0:16])), "/")
		size, err := strconv.Atoi(strings.TrimSpace(string(header[48:58])))
		if err != nil {
			t.Fatal(err)
		}
		data := make([]byte, size)
		if _, err := io.ReadFull(reader, data); err != nil {
			t.Fatal(err)
		}
		out[name] = data
		if size%2 != 0 {
			if _, err := reader.ReadByte(); err != nil {
				t.Fatal(err)
			}
		}
	}
	return out
}

func keys(values map[string][]byte) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	return out
}
