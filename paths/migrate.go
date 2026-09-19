package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/wanstu/wails-desktop-kit/atomicfile"
)

// SamePath compares two filesystem paths after absolute clean normalization.
func SamePath(a, b string) bool {
	aa, errA := filepath.Abs(strings.TrimSpace(a))
	bb, errB := filepath.Abs(strings.TrimSpace(b))
	if errA != nil || errB != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	if isCaseInsensitivePathPlatform() {
		return strings.EqualFold(filepath.Clean(aa), filepath.Clean(bb))
	}
	return filepath.Clean(aa) == filepath.Clean(bb)
}

// MigrateFileIfMissing copies source to target only when target does not exist.
// Missing source is not an error.
func MigrateFileIfMissing(source, target string) (bool, error) {
	if SamePath(source, target) {
		return false, nil
	}
	if _, err := os.Stat(target); err == nil {
		return false, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("paths: inspect migration target: %w", err)
	}
	info, err := os.Stat(source)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("paths: inspect migration source: %w", err)
	}
	if !info.Mode().IsRegular() {
		return false, nil
	}
	if err := atomicfile.Copy(source, target, info.Mode().Perm()); err != nil {
		return false, fmt.Errorf("paths: migrate file: %w", err)
	}
	return true, nil
}

// MigrateFilesIfMissing migrates a bounded explicit list of relative file names.
// It does not recursively copy directories.
func MigrateFilesIfMissing(sourceDir, targetDir string, names ...string) (int, error) {
	count := 0
	for _, name := range names {
		name = filepath.Clean(strings.TrimSpace(name))
		if name == "." || filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
			return count, fmt.Errorf("paths: migration name must stay relative: %q", name)
		}
		ok, err := MigrateFileIfMissing(filepath.Join(sourceDir, name), filepath.Join(targetDir, name))
		if err != nil {
			return count, err
		}
		if ok {
			count++
		}
	}
	return count, nil
}

// MigrateTreeMissing recursively copies regular files that are missing from
// target. Existing target files are never overwritten and symlinks are skipped.
func MigrateTreeMissing(source, target string) (int, error) {
	info, err := os.Lstat(source)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("paths: inspect migration tree source: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return 0, nil
	}
	if info.Mode().IsRegular() {
		ok, err := MigrateFileIfMissing(source, target)
		if err != nil {
			return 0, err
		}
		if ok {
			return 1, nil
		}
		return 0, nil
	}
	if !info.IsDir() {
		return 0, nil
	}
	if err := os.MkdirAll(target, 0o700); err != nil {
		return 0, fmt.Errorf("paths: create migration target dir: %w", err)
	}
	entries, err := os.ReadDir(source)
	if err != nil {
		return 0, fmt.Errorf("paths: read migration source dir: %w", err)
	}
	count := 0
	for _, entry := range entries {
		n, err := MigrateTreeMissing(filepath.Join(source, entry.Name()), filepath.Join(target, entry.Name()))
		if err != nil {
			return count, err
		}
		count += n
	}
	return count, nil
}
