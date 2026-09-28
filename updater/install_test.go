package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVerifyInstallDownloadAcceptsVerifiedSetup(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "demo-v1.2.3-windows-amd64-setup.exe")
	payload := []byte("verified setup payload")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	hash := hex.EncodeToString(sum[:])

	got, err := verifyInstallDownload(DownloadResult{
		Asset: Asset{
			Name:   filepath.Base(path),
			Size:   int64(len(payload)),
			SHA256: hash,
		},
		Path:   path,
		SHA256: hash,
		Bytes:  int64(len(payload)),
	})
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(path)
	if got != filepath.Clean(want) {
		t.Fatalf("verified path = %q want %q", got, filepath.Clean(want))
	}
}

func TestVerifyInstallDownloadRejectsChangedSetup(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "demo-windows-amd64-setup.exe")
	original := []byte("original")
	if err := os.WriteFile(path, original, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(original)
	hash := hex.EncodeToString(sum[:])
	if err := os.WriteFile(path, []byte("tampered!"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := verifyInstallDownload(DownloadResult{
		Asset:  Asset{Name: filepath.Base(path)},
		Path:   path,
		SHA256: hash,
	})
	if !errors.Is(err, ErrUnverifiedDownload) {
		t.Fatalf("expected ErrUnverifiedDownload, got %v", err)
	}
}

func TestVerifyInstallDownloadRejectsNonSetupAsset(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "demo-windows-amd64.exe")
	payload := []byte("portable")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)

	_, err := verifyInstallDownload(DownloadResult{
		Asset:  Asset{Name: filepath.Base(path)},
		Path:   path,
		SHA256: hex.EncodeToString(sum[:]),
	})
	if !errors.Is(err, ErrUnverifiedDownload) {
		t.Fatalf("expected ErrUnverifiedDownload, got %v", err)
	}
}

func TestVerifyInstallDownloadRejectsAssetHashDisagreement(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "demo-windows-amd64-setup.exe")
	payload := []byte("setup")
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(payload)
	hash := hex.EncodeToString(sum[:])

	_, err := verifyInstallDownload(DownloadResult{
		Asset: Asset{
			Name:   filepath.Base(path),
			SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		},
		Path:   path,
		SHA256: hash,
	})
	if !errors.Is(err, ErrUnverifiedDownload) {
		t.Fatalf("expected ErrUnverifiedDownload, got %v", err)
	}
}

func TestInstallAndRestartRejectsUnsupportedOrPortableRuntime(t *testing.T) {
	_, err := InstallAndRestart("desktopkit-test-not-installed", DownloadResult{})
	if runtime.GOOS == "windows" {
		if !errors.Is(err, ErrPortableInstall) {
			t.Fatalf("Windows portable test process should return ErrPortableInstall, got %v", err)
		}
		return
	}
	if !errors.Is(err, ErrInstallUnsupported) {
		t.Fatalf("non-Windows platform should return ErrInstallUnsupported, got %v", err)
	}
}
