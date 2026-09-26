package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/wanstu/wails-desktop-kit/atomicfile"
)

var releaseVersionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:[-+].*)?$`)

type buildMetadata struct {
	Version        string `json:"version"`
	ProductVersion string `json:"product_version"`
	Commit         string `json:"commit,omitempty"`
}

func runBuildMeta(args []string) error {
	if len(args) == 0 || args[0] != "prepare" {
		return fmt.Errorf("buildmeta target is required (supported: prepare)")
	}
	flags := flag.NewFlagSet("desktopkit buildmeta prepare", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var root, desktopDir, version, commit string
	flags.StringVar(&root, "root", ".", "consumer repository root")
	flags.StringVar(&desktopDir, "desktop-dir", ".", "directory containing wails.json")
	flags.StringVar(&version, "version", "dev", "display version, normally the release tag")
	flags.StringVar(&commit, "commit", "", "source commit")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	return prepareBuildMeta(root, desktopDir, version, commit)
}

func prepareBuildMeta(root, desktopDir, version, commit string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	if !filepath.IsAbs(desktopDir) {
		desktopDir = filepath.Join(root, desktopDir)
	}
	version = strings.TrimSpace(version)
	if version == "" {
		version = "dev"
	}
	productVersion := numericProductVersion(version)

	wailsPath := filepath.Join(desktopDir, "wails.json")
	body, err := os.ReadFile(wailsPath)
	if err != nil {
		return fmt.Errorf("read wails.json: %w", err)
	}
	var manifest map[string]any
	if err := json.Unmarshal(body, &manifest); err != nil {
		return fmt.Errorf("parse wails.json: %w", err)
	}
	info, _ := manifest["info"].(map[string]any)
	if info == nil {
		info = map[string]any{}
		manifest["info"] = info
	}
	info["productVersion"] = productVersion
	if err := writeJSON(wailsPath, manifest); err != nil {
		return err
	}

	frontendDir := "frontend"
	if value, ok := manifest["frontend:dir"].(string); ok && strings.TrimSpace(value) != "" {
		frontendDir = value
	}
	if !filepath.IsAbs(frontendDir) {
		frontendDir = filepath.Join(desktopDir, frontendDir)
	}
	meta := buildMetadata{Version: version, ProductVersion: productVersion, Commit: strings.TrimSpace(commit)}
	if err := os.MkdirAll(frontendDir, 0o755); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(frontendDir, "desktopkit-build-info.json"), meta); err != nil {
		return err
	}

	// Wails' fixed Windows file version must stay numeric, while Explorer's
	// ProductVersion string may retain prerelease/build metadata.
	windowsInfoPath := filepath.Join(desktopDir, "build", "windows", "info.json")
	if winBody, readErr := os.ReadFile(windowsInfoPath); readErr == nil {
		var win map[string]any
		if json.Unmarshal(winBody, &win) == nil {
			fixed, _ := win["fixed"].(map[string]any)
			if fixed == nil {
				fixed = map[string]any{}
				win["fixed"] = fixed
			}
			fixed["file_version"] = productVersion
			infoRoot, _ := win["info"].(map[string]any)
			if infoRoot != nil {
				if lang, ok := infoRoot["0000"].(map[string]any); ok {
					lang["ProductVersion"] = strings.TrimPrefix(version, "v")
				}
			}
			if err := writeJSON(windowsInfoPath, win); err != nil {
				return err
			}
		}
	}

	fmt.Printf("build metadata: version=%s productVersion=%s commit=%s\n", version, productVersion, shortCommit(commit))
	return nil
}

func numericProductVersion(version string) string {
	match := releaseVersionPattern.FindStringSubmatch(strings.TrimSpace(version))
	if len(match) == 4 {
		return match[1] + "." + match[2] + "." + match[3]
	}
	return "0.0.0"
}

func shortCommit(commit string) string {
	commit = strings.TrimSpace(commit)
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}

func writeJSON(path string, value any) error {
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return atomicfile.Write(path, body, 0o644)
}
