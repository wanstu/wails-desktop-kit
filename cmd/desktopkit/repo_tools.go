package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/wanstu/wails-desktop-kit/atomicfile"
)

var (
	kitRequirePattern    = regexp.MustCompile("(?m)^\\s*github\\.com/wanstu/wails-desktop-kit\\s+(v[^\\s]+)")
	wailsRequirePattern  = regexp.MustCompile("(?m)^\\s*github\\.com/wailsapp/wails/v2\\s+(v[^\\s]+)")
	workflowRefPattern   = regexp.MustCompile("wanstu/wails-desktop-kit/\\.github/workflows/wails-desktop\\.yml@([^\\s\\\"'#]+)")
	appNamePatternDoctor = regexp.MustCompile("(?m)^\\s*app-name:\\s*[\\\"']?([^\\s\\\"'#]+)")
	versionPattern       = regexp.MustCompile("^v[0-9]+\\.[0-9]+\\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$")
)

type repoInspection struct {
	KitModuleVersion string
	WailsVersion     string
	WorkflowVersions map[string]string
	WailsOutputs     map[string]string
	WorkflowAppNames map[string][]string
}

func runDoctor(args []string) error {
	flags := flag.NewFlagSet("desktopkit doctor", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var root string
	flags.StringVar(&root, "root", ".", "consumer repository root")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	inspection, err := inspectRepository(root)
	if err != nil {
		return err
	}
	fmt.Printf("root: %s\n", root)
	fmt.Printf("desktop-kit module: %s\n", valueOrDash(inspection.KitModuleVersion))
	fmt.Printf("Wails module: %s\n", valueOrDash(inspection.WailsVersion))
	for _, path := range sortedKeys(inspection.WorkflowVersions) {
		fmt.Printf("workflow: %s -> %s\n", path, inspection.WorkflowVersions[path])
	}
	for _, path := range sortedKeys(inspection.WailsOutputs) {
		fmt.Printf("wails: %s -> outputfilename=%s\n", path, inspection.WailsOutputs[path])
	}

	var issues []string
	if inspection.KitModuleVersion == "" {
		issues = append(issues, "go.mod does not require github.com/wanstu/wails-desktop-kit")
	}
	versions := map[string]struct{}{}
	for _, version := range inspection.WorkflowVersions {
		versions[version] = struct{}{}
		if inspection.KitModuleVersion != "" && version != inspection.KitModuleVersion {
			issues = append(issues, fmt.Sprintf("workflow %s differs from Go module %s", version, inspection.KitModuleVersion))
		}
	}
	if len(versions) > 1 {
		issues = append(issues, "reusable workflow references use multiple Kit versions")
	}
	for workflow, names := range inspection.WorkflowAppNames {
		for _, name := range names {
			found := false
			for _, output := range inspection.WailsOutputs {
				if output == name {
					found = true
					break
				}
			}
			if len(inspection.WailsOutputs) > 0 && !found {
				issues = append(issues, fmt.Sprintf("%s app-name %q does not match any wails.json outputfilename", workflow, name))
			}
		}
	}
	if len(issues) == 0 {
		fmt.Println("doctor: OK")
		return nil
	}
	for _, issue := range issues {
		fmt.Println("doctor: ISSUE:", issue)
	}
	return fmt.Errorf("doctor found %d issue(s)", len(issues))
}

func runUpgrade(args []string) error {
	flags := flag.NewFlagSet("desktopkit upgrade", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	var root, version string
	var tidy bool
	flags.StringVar(&root, "root", ".", "consumer repository root")
	flags.StringVar(&version, "to", "", "target Kit version, for example v0.8.0")
	flags.BoolVar(&tidy, "tidy", true, "run go mod tidy after updating go.mod")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("--to must be a semantic version like v0.8.0")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	goMod := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(goMod)
	if err != nil {
		return fmt.Errorf("read go.mod: %w", err)
	}
	if !kitRequirePattern.Match(data) {
		return fmt.Errorf("go.mod does not require github.com/wanstu/wails-desktop-kit")
	}
	updated := kitRequirePattern.ReplaceAll(data, []byte("github.com/wanstu/wails-desktop-kit "+version))
	if !bytes.Equal(updated, data) {
		if err := atomicfile.Write(goMod, updated, 0o644); err != nil {
			return err
		}
		fmt.Printf("updated go.mod -> %s\n", version)
	}
	workflowRoot := filepath.Join(root, ".github", "workflows")
	if err := filepath.WalkDir(workflowRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		next := workflowRefPattern.ReplaceAll(body, []byte("wanstu/wails-desktop-kit/.github/workflows/wails-desktop.yml@"+version))
		if bytes.Equal(next, body) {
			return nil
		}
		if err := atomicfile.Write(path, next, 0o644); err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		fmt.Printf("updated %s -> %s\n", filepath.ToSlash(rel), version)
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return err
	}
	if tidy {
		command := exec.Command("go", "mod", "tidy")
		command.Dir = root
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		if err := command.Run(); err != nil {
			return fmt.Errorf("go mod tidy: %w", err)
		}
	}
	return nil
}

func inspectRepository(root string) (repoInspection, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return repoInspection{}, err
	}
	out := repoInspection{
		WorkflowVersions: map[string]string{},
		WailsOutputs:     map[string]string{},
		WorkflowAppNames: map[string][]string{},
	}
	goMod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return out, fmt.Errorf("read go.mod: %w", err)
	}
	if match := kitRequirePattern.FindSubmatch(goMod); len(match) == 2 {
		out.KitModuleVersion = string(match[1])
	}
	if match := wailsRequirePattern.FindSubmatch(goMod); len(match) == 2 {
		out.WailsVersion = string(match[1])
	}

	workflowRoot := filepath.Join(root, ".github", "workflows")
	if err := filepath.WalkDir(workflowRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if match := workflowRefPattern.FindSubmatch(body); len(match) == 2 {
			out.WorkflowVersions[rel] = string(match[1])
		}
		matches := appNamePatternDoctor.FindAllSubmatch(body, -1)
		for _, match := range matches {
			if len(match) == 2 && !bytes.Contains(match[1], []byte("$"+"{{")) {
				out.WorkflowAppNames[rel] = append(out.WorkflowAppNames[rel], string(match[1]))
			}
		}
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return out, err
	}

	if err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", ".next", "dist":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if entry.Name() != "wails.json" {
			return nil
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var manifest struct {
			OutputFilename string
		}
		if json.Unmarshal(body, &manifest) != nil || strings.TrimSpace(manifest.OutputFilename) == "" {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		out.WailsOutputs[filepath.ToSlash(rel)] = strings.TrimSpace(manifest.OutputFilename)
		return nil
	}); err != nil {
		return out, err
	}
	return out, nil
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func valueOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}
