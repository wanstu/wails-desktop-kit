package packaging

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
)

func TestDebDataTarHasParentDirectoriesBeforeFiles(t *testing.T) {
	root := t.TempDir()
	input := filepath.Join(root, "demo")
	desktop := filepath.Join(root, "demo.desktop")
	icon := filepath.Join(root, "demo.png")
	appstream := filepath.Join(root, "demo.metainfo.xml")
	copyright := filepath.Join(root, "copyright")
	for name, content := range map[string]string{
		input: "demo binary", desktop: "[Desktop Entry]\nName=Demo\n", icon: "demo icon",
		appstream: "<component><id>io.example.Demo</id></component>",
		copyright: "Example license text",
	} {
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	request := LinuxRequest{
		Input: input, AppName: "demo", PackageName: "demo", Description: "Demo service",
		DesktopFile: desktop, IconFile: icon, AppStreamFile: appstream,
		CopyrightFile: copyright, Systemd: &LinuxSystemdService{},
	}
	data, err := buildDataTarGz(request)
	if err != nil {
		t.Fatal(err)
	}
	again, err := buildDataTarGz(request)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, again) {
		t.Fatal("data.tar.gz output must be reproducible")
	}

	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	reader := tar.NewReader(gz)
	seenDirs := map[string]bool{}
	seenFiles := map[string]bool{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Typeflag == tar.TypeDir {
			if !strings.HasSuffix(header.Name, "/") || header.Size != 0 || header.Mode != 0o755 {
				t.Fatalf("invalid directory header: %#v", header)
			}
			seenDirs[header.Name] = true
			continue
		}
		if header.Typeflag != tar.TypeReg {
			t.Fatalf("unexpected entry type %c for %s", header.Typeflag, header.Name)
		}
		seenFiles[header.Name] = true
		clean := strings.TrimPrefix(header.Name, "./")
		for parent := path.Dir(clean); parent != "." && parent != "/"; parent = path.Dir(parent) {
			if !seenDirs["./"+parent+"/"] {
				t.Fatalf("missing parent directory before file %q: %q", header.Name, parent)
			}
		}
	}
	for _, want := range []string{
		"./usr/bin/demo",
		"./usr/share/applications/demo.desktop",
		"./usr/share/icons/hicolor/256x256/apps/demo.png",
		"./lib/systemd/system/demo.service",
		"./usr/share/metainfo/io.example.Demo.metainfo.xml",
		"./usr/share/doc/demo/copyright",
	} {
		if !seenFiles[want] {
			t.Fatalf("missing expected file %q", want)
		}
	}
}
