package updater

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type staticProvider struct {
	release Release
	err     error
}

func (provider staticProvider) Latest(context.Context) (Release, error) {
	return provider.release, provider.err
}

func TestCheckDoesNotRequireAssetWhenCurrent(t *testing.T) {
	client := Client{
		CurrentVersion: "v1.2.3",
		Provider:       staticProvider{release: Release{Version: "v1.2.3"}},
		SelectAsset:    AssetBySuffix("missing"),
	}
	result, err := client.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.UpdateAvailable {
		t.Fatalf("unexpected update: %+v", result)
	}
}

func TestCheckUsesProviderDigest(t *testing.T) {
	hash := strings.Repeat("a", 64)
	client := Client{
		CurrentVersion: "v1.0.0",
		Provider: staticProvider{release: Release{
			Version: "v1.1.0",
			Assets: []Asset{{
				Name:   "demo-v1.1.0-windows-amd64-setup.exe",
				URL:    "https://example.test/demo.exe",
				SHA256: "sha256:" + hash,
			}},
		}},
		SelectAsset: AssetBySuffix("windows-amd64-setup.exe"),
	}
	result, err := client.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !result.UpdateAvailable || result.Asset.SHA256 != hash || result.ChecksumAsset.Name != "" {
		t.Fatalf("result = %+v", result)
	}
}

func TestCheckUsesSidecarChecksum(t *testing.T) {
	client := Client{
		CurrentVersion: "1.0.0",
		Provider: staticProvider{release: Release{
			Version: "1.1.0",
			Assets: []Asset{
				{Name: "demo-v1.1.0-setup.exe", URL: "https://example.test/demo.exe"},
				{Name: "demo-v1.1.0-setup.exe.sha256", URL: "https://example.test/demo.sha256"},
			},
		}},
		SelectAsset: AssetBySuffix("-setup.exe"),
	}
	result, err := client.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.ChecksumAsset.Name != "demo-v1.1.0-setup.exe.sha256" {
		t.Fatalf("checksum asset = %+v", result.ChecksumAsset)
	}
}

func TestCheckRejectsMissingChecksum(t *testing.T) {
	client := Client{
		CurrentVersion: "1.0.0",
		Provider: staticProvider{release: Release{
			Version: "1.1.0",
			Assets:  []Asset{{Name: "demo-setup.exe", URL: "https://example.test/demo.exe"}},
		}},
		SelectAsset: AssetBySuffix("-setup.exe"),
	}
	_, err := client.Check(context.Background())
	if !errors.Is(err, ErrChecksumMissing) {
		t.Fatalf("expected ErrChecksumMissing, got %v", err)
	}
}

func TestDownloadVerifiesSidecarAndReportsProgress(t *testing.T) {
	payload := []byte("verified update payload")
	sum := sha256.Sum256(payload)
	expected := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/asset":
			_, _ = w.Write(payload)
		case "/asset.sha256":
			fmt.Fprintf(w, "%s  demo-setup.exe\n", expected)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := Client{HTTPClient: server.Client()}
	var progress []Progress
	result, err := client.Download(context.Background(), CheckResult{
		UpdateAvailable: true,
		Asset: Asset{
			Name: "demo-setup.exe",
			URL:  server.URL + "/asset",
			Size: int64(len(payload)),
		},
		ChecksumAsset: Asset{
			Name: "demo-setup.exe.sha256",
			URL:  server.URL + "/asset.sha256",
		},
	}, t.TempDir(), func(value Progress) {
		progress = append(progress, value)
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SHA256 != expected || result.Bytes != int64(len(payload)) {
		t.Fatalf("result = %+v", result)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(payload) {
		t.Fatalf("downloaded payload = %q", string(data))
	}
	if len(progress) < 2 {
		t.Fatalf("progress callbacks = %+v", progress)
	}
	last := progress[len(progress)-1]
	if last.Downloaded != int64(len(payload)) || last.Total != int64(len(payload)) {
		t.Fatalf("last progress = %+v", last)
	}
}

func TestDownloadRejectsChecksumMismatchWithoutPublishingFile(t *testing.T) {
	payload := []byte("bad payload")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	dir := t.TempDir()
	client := Client{HTTPClient: server.Client()}
	_, err := client.Download(context.Background(), CheckResult{
		UpdateAvailable: true,
		Asset: Asset{
			Name:   "demo-setup.exe",
			URL:    server.URL,
			Size:   int64(len(payload)),
			SHA256: strings.Repeat("0", 64),
		},
	}, dir, nil)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum mismatch, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "demo-setup.exe")); !os.IsNotExist(statErr) {
		t.Fatalf("final update file should not exist, stat err=%v", statErr)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("temporary files remained: %+v", entries)
	}
}

func TestDownloadRejectsExistingDestination(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo.exe")
	if err := os.WriteFile(path, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	client := Client{}
	_, err := client.Download(context.Background(), CheckResult{
		UpdateAvailable: true,
		Asset: Asset{
			Name:   "demo.exe",
			URL:    "https://example.test/demo.exe",
			SHA256: strings.Repeat("a", 64),
		},
	}, dir, nil)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected existing destination error, got %v", err)
	}
}
