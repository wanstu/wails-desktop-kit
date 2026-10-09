package main

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/wanstu/wails-desktop-kit/atomicfile"
)

const kitScriptTagRE = `v[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?`

var (
	kitToolScriptRef     = regexp.MustCompile(`github\.com/wanstu/wails-desktop-kit/cmd/desktopkit@` + kitScriptTagRE)
	kitWorkflowHelperRef = regexp.MustCompile(`(?m)^\s*desktopkit-cli-version:\s*["']?` + kitScriptTagRE)
	kitTagInRef          = regexp.MustCompile(kitScriptTagRE)
)

// Kit tool pins appear in build scripts as well as workflow uses. A dynamic
// version (for example @${{ inputs.version }}) is not a fixed pin and is
// intentionally left unchanged.
func scriptKitPins(body []byte) []string {
	seen := make(map[string]bool)
	for _, pattern := range []*regexp.Regexp{kitToolScriptRef, kitWorkflowHelperRef} {
		for _, ref := range pattern.FindAll(body, -1) {
			version := kitTagInRef.Find(ref)
			if len(version) != 0 {
				seen[string(version)] = true
			}
		}
	}
	versions := make([]string, 0, len(seen))
	for ver := range seen {
		versions = append(versions, ver)
	}
	sort.Strings(versions)
	return versions
}

func rewriteScriptKitPins(body []byte, target string) []byte {
	out := body
	for _, pattern := range []*regexp.Regexp{kitToolScriptRef, kitWorkflowHelperRef} {
		out = pattern.ReplaceAllFunc(out, func(ref []byte) []byte {
			location := kitTagInRef.FindIndex(ref)
			if location == nil {
				return ref
			}
			changed := make([]byte, 0, len(ref)-location[1]+location[0]+len(target))
			changed = append(changed, ref[:location[0]]...)
			changed = append(changed, target...)
			changed = append(changed, ref[location[1]:]...)
			return changed
		})
	}
	return out
}

func walkKitScriptFiles(root string, visit func(path string, body []byte) error) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root {
				switch strings.ToLower(entry.Name()) {
				case ".git", ".adm", "node_modules", "vendor", ".next", "dist", "docs", ".venv", ".cache":
					return filepath.SkipDir
				}
			}
			return nil
		}
		name := strings.ToLower(entry.Name())
		ext := strings.ToLower(filepath.Ext(name))
		switch ext {
		case ".sh", ".ps1", ".bat", ".cmd", ".yml", ".yaml", ".mk":
		default:
			if name != "makefile" && name != "dockerfile" && name != "justfile" {
				return nil
			}
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > 2*1024*1024 {
			return fmt.Errorf("build script exceeds inspection limit: %s", path)
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return visit(path, body)
	})
}

func inspectScriptKitPins(root string) (map[string][]string, error) {
	pins := make(map[string][]string)
	err := walkKitScriptFiles(root, func(path string, body []byte) error {
		versions := scriptKitPins(body)
		if len(versions) == 0 {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		pins[filepath.ToSlash(rel)] = versions
		return nil
	})
	return pins, err
}

func upgradeScriptKitPins(root, target string) error {
	return walkKitScriptFiles(root, func(path string, body []byte) error {
		updated := rewriteScriptKitPins(body, target)
		if bytes.Equal(body, updated) {
			return nil
		}
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if err := atomicfile.Write(path, updated, info.Mode().Perm()); err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		fmt.Printf("updated Kit tool pin in %s -> %s\n", filepath.ToSlash(rel), target)
		return nil
	})
}
