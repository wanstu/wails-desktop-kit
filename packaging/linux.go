package packaging

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Format string

const (
	FormatRaw   Format = "raw"
	FormatDeb   Format = "deb"
	FormatTarGz Format = "tar.gz"
)

var (
	packageNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9+.-]*$`)
	appNamePattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)
)

type LinuxBinary struct {
	Source      string
	InstallName string
}

type LinuxRequest struct {
	Input          string
	OutputDir      string
	AppName        string
	AssetBase      string
	PackageName    string
	PackageVersion string
	Architecture   string
	Description    string
	Maintainer     string
	Section        string
	Priority       string
	Depends        string
	DesktopFile    string
	IconFile       string
	ExtraBinaries  []LinuxBinary
	Formats        []Format
}

type Artifact struct {
	Path   string
	Format Format
	SHA256 string
	Size   int64
}

type linuxBuilder interface {
	Format() Format
	Build(LinuxRequest) (string, error)
}

type linuxRawBuilder struct{}
type linuxTarGzBuilder struct{}
type linuxDebBuilder struct{}

func ParseFormats(value string) ([]Format, error) {
	seen := map[Format]bool{}
	var out []Format
	for _, raw := range strings.Split(value, ",") {
		name := Format(strings.TrimSpace(strings.ToLower(raw)))
		if name == "" {
			continue
		}
		switch name {
		case FormatRaw, FormatDeb, FormatTarGz:
		default:
			return nil, fmt.Errorf("unsupported package format %q (supported: raw, deb, tar.gz)", name)
		}
		if !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	if len(out) == 0 {
		return []Format{FormatRaw}, nil
	}
	return out, nil
}

func PackageLinux(request LinuxRequest) ([]Artifact, error) {
	request = normalizeLinuxRequest(request)
	if err := validateLinuxRequest(request); err != nil {
		return nil, err
	}
	builders := map[Format]linuxBuilder{
		FormatRaw:   linuxRawBuilder{},
		FormatTarGz: linuxTarGzBuilder{},
		FormatDeb:   linuxDebBuilder{},
	}
	if err := os.MkdirAll(request.OutputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	artifacts := make([]Artifact, 0, len(request.Formats))
	for _, format := range request.Formats {
		builder := builders[format]
		if builder == nil {
			return nil, fmt.Errorf("unsupported package format %q", format)
		}
		path, err := builder.Build(request)
		if err != nil {
			return nil, fmt.Errorf("package %s: %w", format, err)
		}
		artifact, err := checksumArtifact(path, format)
		if err != nil {
			return nil, err
		}
		artifacts = append(artifacts, artifact)
	}
	sort.SliceStable(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	return artifacts, nil
}

func normalizeLinuxRequest(request LinuxRequest) LinuxRequest {
	request.Input = filepath.Clean(request.Input)
	request.OutputDir = filepath.Clean(request.OutputDir)
	request.AppName = strings.TrimSpace(request.AppName)
	request.AssetBase = strings.TrimSpace(request.AssetBase)
	request.PackageName = strings.TrimSpace(strings.ToLower(request.PackageName))
	request.PackageVersion = strings.TrimSpace(request.PackageVersion)
	request.Architecture = strings.TrimSpace(request.Architecture)
	request.Description = strings.TrimSpace(request.Description)
	request.Maintainer = strings.TrimSpace(request.Maintainer)
	request.Section = strings.TrimSpace(request.Section)
	request.Priority = strings.TrimSpace(request.Priority)
	request.Depends = strings.TrimSpace(request.Depends)
	request.DesktopFile = strings.TrimSpace(request.DesktopFile)
	request.IconFile = strings.TrimSpace(request.IconFile)
	for i := range request.ExtraBinaries {
		request.ExtraBinaries[i].Source = filepath.Clean(strings.TrimSpace(request.ExtraBinaries[i].Source))
		request.ExtraBinaries[i].InstallName = strings.TrimSpace(request.ExtraBinaries[i].InstallName)
	}
	if request.PackageName == "" {
		request.PackageName = normalizePackageName(request.AppName)
	}
	if request.PackageVersion == "" {
		request.PackageVersion = "0.0.0"
	}
	if request.Architecture == "" {
		request.Architecture = "amd64"
	}
	if request.Description == "" {
		request.Description = "Wails desktop application"
	}
	if request.Maintainer == "" {
		request.Maintainer = "unknown"
	}
	if request.Section == "" {
		request.Section = "utils"
	}
	if request.Priority == "" {
		request.Priority = "optional"
	}
	if len(request.Formats) == 0 {
		request.Formats = []Format{FormatRaw}
	}
	return request
}

func validateLinuxRequest(request LinuxRequest) error {
	if request.Input == "" || request.Input == "." {
		return errors.New("input binary is required")
	}
	info, err := os.Stat(request.Input)
	if err != nil {
		return fmt.Errorf("stat input binary: %w", err)
	}
	if !info.Mode().IsRegular() {
		return errors.New("input must be a regular file")
	}
	if strings.TrimSpace(request.OutputDir) == "" {
		return errors.New("output dir is required")
	}
	if !appNamePattern.MatchString(request.AppName) {
		return fmt.Errorf("invalid app name %q", request.AppName)
	}
	if request.AssetBase == "" || strings.ContainsAny(request.AssetBase, `/\`) {
		return fmt.Errorf("invalid asset base %q", request.AssetBase)
	}
	if !packageNamePattern.MatchString(request.PackageName) {
		return fmt.Errorf("invalid Debian package name %q", request.PackageName)
	}
	if strings.ContainsAny(request.PackageVersion, "\r\n") || request.PackageVersion == "" {
		return errors.New("invalid package version")
	}
	if request.Architecture != "amd64" && request.Architecture != "arm64" {
		return fmt.Errorf("unsupported architecture %q", request.Architecture)
	}
	for _, optional := range []struct {
		name string
		path string
	}{
		{"desktop file", request.DesktopFile},
		{"icon file", request.IconFile},
	} {
		if optional.path == "" {
			continue
		}
		info, err := os.Stat(optional.path)
		if err != nil {
			return fmt.Errorf("stat %s: %w", optional.name, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s must be a regular file", optional.name)
		}
	}
	names := map[string]struct{}{request.AppName: {}}
	for _, binary := range request.ExtraBinaries {
		if binary.Source == "" || binary.Source == "." {
			return errors.New("extra binary source is required")
		}
		if !appNamePattern.MatchString(binary.InstallName) {
			return fmt.Errorf("invalid extra binary install name %q", binary.InstallName)
		}
		if _, exists := names[binary.InstallName]; exists {
			return fmt.Errorf("duplicate installed binary name %q", binary.InstallName)
		}
		names[binary.InstallName] = struct{}{}
		info, err := os.Stat(binary.Source)
		if err != nil {
			return fmt.Errorf("stat extra binary %q: %w", binary.InstallName, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("extra binary %q must be a regular file", binary.InstallName)
		}
	}
	for _, format := range request.Formats {
		switch format {
		case FormatRaw, FormatDeb, FormatTarGz:
		default:
			return fmt.Errorf("unsupported package format %q", format)
		}
	}
	return nil
}

func normalizePackageName(name string) string {
	name = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "_", "-"))
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '+' || r == '.' || r == '-' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-.")
}

func (linuxRawBuilder) Format() Format { return FormatRaw }
func (linuxRawBuilder) Build(request LinuxRequest) (string, error) {
	target := filepath.Join(request.OutputDir, request.AssetBase+"-linux-"+request.Architecture)
	if err := copyFile(request.Input, target, 0o755); err != nil {
		return "", err
	}
	return target, nil
}

func (linuxTarGzBuilder) Format() Format { return FormatTarGz }
func (linuxTarGzBuilder) Build(request LinuxRequest) (string, error) {
	target := filepath.Join(request.OutputDir, request.AssetBase+"-linux-"+request.Architecture+".tar.gz")
	file, err := os.Create(target)
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(target)
		}
	}()

	gz := gzip.NewWriter(file)
	tw := tar.NewWriter(gz)
	root := request.AssetBase + "-linux-" + request.Architecture

	if err := addTarFile(tw, request.Input, filepath.ToSlash(filepath.Join(root, request.AppName)), 0o755); err != nil {
		return "", err
	}
	for _, binary := range request.ExtraBinaries {
		if err := addTarFile(tw, binary.Source, filepath.ToSlash(filepath.Join(root, binary.InstallName)), 0o755); err != nil {
			return "", err
		}
	}
	if request.DesktopFile != "" {
		if err := addTarFile(tw, request.DesktopFile, filepath.ToSlash(filepath.Join(root, request.PackageName+".desktop")), 0o644); err != nil {
			return "", err
		}
	}
	if request.IconFile != "" {
		ext := filepath.Ext(request.IconFile)
		if ext == "" {
			ext = ".png"
		}
		if err := addTarFile(tw, request.IconFile, filepath.ToSlash(filepath.Join(root, request.PackageName+ext)), 0o644); err != nil {
			return "", err
		}
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	if err := gz.Close(); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	ok = true
	return target, nil
}

func (linuxDebBuilder) Format() Format { return FormatDeb }
func (linuxDebBuilder) Build(request LinuxRequest) (string, error) {
	target := filepath.Join(request.OutputDir, request.AssetBase+"-linux-"+request.Architecture+".deb")
	control, err := buildControlTarGz(request)
	if err != nil {
		return "", err
	}
	data, err := buildDataTarGz(request)
	if err != nil {
		return "", err
	}
	file, err := os.Create(target)
	if err != nil {
		return "", err
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(target)
		}
	}()

	if _, err := io.WriteString(file, "!<arch>\n"); err != nil {
		return "", err
	}
	now := time.Now().Unix()
	for _, entry := range []struct {
		name string
		data []byte
	}{
		{"debian-binary", []byte("2.0\n")},
		{"control.tar.gz", control},
		{"data.tar.gz", data},
	} {
		if err := writeArEntry(file, entry.name, entry.data, now); err != nil {
			return "", err
		}
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	ok = true
	return target, nil
}

func buildControlTarGz(request LinuxRequest) ([]byte, error) {
	description := strings.ReplaceAll(request.Description, "\r\n", "\n")
	description = strings.ReplaceAll(description, "\r", "\n")
	lines := strings.Split(description, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		lines = []string{"Wails desktop application"}
	}
	var control strings.Builder
	fmt.Fprintf(&control, "Package: %s\n", request.PackageName)
	fmt.Fprintf(&control, "Version: %s\n", request.PackageVersion)
	fmt.Fprintf(&control, "Section: %s\n", request.Section)
	fmt.Fprintf(&control, "Priority: %s\n", request.Priority)
	fmt.Fprintf(&control, "Architecture: %s\n", request.Architecture)
	fmt.Fprintf(&control, "Maintainer: %s\n", strings.ReplaceAll(request.Maintainer, "\n", " "))
	if request.Depends != "" {
		fmt.Fprintf(&control, "Depends: %s\n", strings.ReplaceAll(request.Depends, "\n", " "))
	}
	fmt.Fprintf(&control, "Description: %s\n", strings.TrimSpace(lines[0]))
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			control.WriteString(" .\n")
		} else {
			fmt.Fprintf(&control, " %s\n", strings.TrimSpace(line))
		}
	}
	return tarGzBytes([]tarEntry{{name: "./control", data: []byte(control.String()), mode: 0o644}})
}

func buildDataTarGz(request LinuxRequest) ([]byte, error) {
	binary, err := os.ReadFile(request.Input)
	if err != nil {
		return nil, err
	}
	entries := []tarEntry{{
		name: "./usr/bin/" + request.AppName,
		data: binary,
		mode: 0o755,
	}}
	for _, extra := range request.ExtraBinaries {
		data, err := os.ReadFile(extra.Source)
		if err != nil {
			return nil, err
		}
		entries = append(entries, tarEntry{
			name: "./usr/bin/" + extra.InstallName,
			data: data,
			mode: 0o755,
		})
	}
	if request.DesktopFile != "" {
		data, err := os.ReadFile(request.DesktopFile)
		if err != nil {
			return nil, err
		}
		entries = append(entries, tarEntry{
			name: "./usr/share/applications/" + request.PackageName + ".desktop",
			data: data,
			mode: 0o644,
		})
	}
	if request.IconFile != "" {
		data, err := os.ReadFile(request.IconFile)
		if err != nil {
			return nil, err
		}
		ext := filepath.Ext(request.IconFile)
		if ext == "" {
			ext = ".png"
		}
		entries = append(entries, tarEntry{
			name: "./usr/share/icons/hicolor/256x256/apps/" + request.PackageName + ext,
			data: data,
			mode: 0o644,
		})
	}
	return tarGzBytes(entries)
}

type tarEntry struct {
	name string
	data []byte
	mode int64
}

func tarGzBytes(entries []tarEntry) ([]byte, error) {
	var buffer bytes.Buffer
	gz := gzip.NewWriter(&buffer)
	tw := tar.NewWriter(gz)
	for _, entry := range entries {
		header := &tar.Header{
			Name:    entry.name,
			Mode:    entry.mode,
			Size:    int64(len(entry.data)),
			ModTime: time.Unix(0, 0),
			Uid:     0,
			Gid:     0,
		}
		if err := tw.WriteHeader(header); err != nil {
			return nil, err
		}
		if _, err := tw.Write(entry.data); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func addTarFile(tw *tar.Writer, source, name string, mode int64) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	header := &tar.Header{
		Name:    name,
		Mode:    mode,
		Size:    int64(len(data)),
		ModTime: time.Unix(0, 0),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err = tw.Write(data)
	return err
}

func writeArEntry(w io.Writer, name string, data []byte, mtime int64) error {
	if len(name) > 15 {
		return fmt.Errorf("ar entry name too long: %s", name)
	}
	header := fmt.Sprintf("%-16s%-12d%-6d%-6d%-8o%-10d`\n", name+"/", mtime, 0, 0, 0o100644, len(data))
	if len(header) != 60 {
		return fmt.Errorf("invalid ar header length for %s: %d", name, len(header))
	}
	if _, err := io.WriteString(w, header); err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if len(data)%2 != 0 {
		_, err := io.WriteString(w, "\n")
		return err
	}
	return nil
}

func checksumArtifact(path string, format Format) (Artifact, error) {
	file, err := os.Open(path)
	if err != nil {
		return Artifact{}, err
	}
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	closeErr := file.Close()
	if err != nil {
		return Artifact{}, err
	}
	if closeErr != nil {
		return Artifact{}, closeErr
	}
	sum := hex.EncodeToString(hash.Sum(nil))
	name := filepath.Base(path)
	if err := os.WriteFile(path+".sha256", []byte(sum+"  "+name+"\n"), 0o644); err != nil {
		return Artifact{}, err
	}
	return Artifact{Path: path, Format: format, SHA256: sum, Size: size}, nil
}

func copyFile(source, target string, mode os.FileMode) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if err := os.WriteFile(target, data, mode); err != nil {
		return err
	}
	return os.Chmod(target, mode)
}
