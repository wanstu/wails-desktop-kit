package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrInstallUnsupported = errors.New("automatic update installation is not supported on this platform")
	ErrPortableInstall    = errors.New("portable applications cannot install updates automatically")
	ErrMachineInstall     = errors.New("machine-scope updates require manual installation")
	ErrUnverifiedDownload = errors.New("update download is not verified")
)

type InstallLaunch struct {
	InstallerPID int
	Installation Installation
	SetupPath    string
	Restart      bool
}

// InstallAndRestart launches a previously verified Kit Windows Setup and returns
// as soon as the installer process starts. The caller must then exit the current
// application normally. The Setup waits for the current PID, installs the update,
// and restarts the user-scope application after installation succeeds.
func InstallAndRestart(appID string, download DownloadResult) (InstallLaunch, error) {
	if !automaticInstallSupported() {
		return InstallLaunch{}, ErrInstallUnsupported
	}

	installation, err := CurrentInstallation(appID)
	if err != nil {
		return InstallLaunch{}, fmt.Errorf("detect current installation: %w", err)
	}
	if !installation.Managed() {
		return InstallLaunch{}, ErrPortableInstall
	}
	if installation.Mode != InstallationUser {
		return InstallLaunch{}, ErrMachineInstall
	}

	setupPath, err := verifyInstallDownload(download)
	if err != nil {
		return InstallLaunch{}, err
	}
	pid, err := launchUpdateSetup(setupPath, os.Getpid())
	if err != nil {
		return InstallLaunch{}, fmt.Errorf("launch update installer: %w", err)
	}
	return InstallLaunch{
		InstallerPID: pid,
		Installation: installation,
		SetupPath:    setupPath,
		Restart:      true,
	}, nil
}

func verifyInstallDownload(download DownloadResult) (string, error) {
	name := strings.TrimSpace(download.Asset.Name)
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return "", fmt.Errorf("%w: invalid asset name %q", ErrUnverifiedDownload, name)
	}
	if !strings.HasSuffix(strings.ToLower(name), "-setup.exe") {
		return "", fmt.Errorf("%w: asset %q is not a Windows setup executable", ErrUnverifiedDownload, name)
	}

	path := strings.TrimSpace(download.Path)
	if path == "" {
		return "", fmt.Errorf("%w: download path is empty", ErrUnverifiedDownload)
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("%w: resolve update path: %v", ErrUnverifiedDownload, err)
	}
	absolute = filepath.Clean(absolute)
	if !strings.EqualFold(filepath.Base(absolute), name) {
		return "", fmt.Errorf("%w: asset name %q does not match file %q", ErrUnverifiedDownload, name, filepath.Base(absolute))
	}

	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("%w: stat update setup: %v", ErrUnverifiedDownload, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Size() <= 0 {
		return "", fmt.Errorf("%w: update setup must be a non-empty regular file", ErrUnverifiedDownload)
	}
	if download.Bytes > 0 && info.Size() != download.Bytes {
		return "", fmt.Errorf("%w: downloaded size changed: got %d want %d", ErrUnverifiedDownload, info.Size(), download.Bytes)
	}
	if download.Asset.Size > 0 && info.Size() != download.Asset.Size {
		return "", fmt.Errorf("%w: asset size changed: got %d want %d", ErrUnverifiedDownload, info.Size(), download.Asset.Size)
	}

	expected, ok := normalizeSHA256(download.SHA256)
	if !ok {
		return "", fmt.Errorf("%w: invalid downloaded SHA256", ErrUnverifiedDownload)
	}
	if assetHash, hasAssetHash := normalizeSHA256(download.Asset.SHA256); hasAssetHash && assetHash != expected {
		return "", fmt.Errorf("%w: asset SHA256 does not match verified download", ErrUnverifiedDownload)
	}

	file, err := os.Open(absolute)
	if err != nil {
		return "", fmt.Errorf("%w: open update setup: %v", ErrUnverifiedDownload, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("%w: hash update setup: %v", ErrUnverifiedDownload, err)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return "", fmt.Errorf("%w: setup SHA256 changed: got %s want %s", ErrUnverifiedDownload, actual, expected)
	}
	return absolute, nil
}
