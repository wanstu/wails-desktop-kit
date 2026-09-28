package updater

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Client struct {
	CurrentVersion string
	Provider       Provider
	SelectAsset    AssetSelector
	HTTPClient     *http.Client
}

func (client *Client) Check(ctx context.Context) (CheckResult, error) {
	if client.Provider == nil {
		return CheckResult{}, errors.New("update provider is required")
	}
	if client.SelectAsset == nil {
		return CheckResult{}, errors.New("update asset selector is required")
	}
	current := strings.TrimSpace(client.CurrentVersion)
	if _, err := parseVersion(current); err != nil {
		return CheckResult{}, fmt.Errorf("invalid current version: %w", err)
	}

	release, err := client.Provider.Latest(ctx)
	if err != nil {
		return CheckResult{}, fmt.Errorf("load latest release: %w", err)
	}
	comparison, err := CompareVersions(release.Version, current)
	if err != nil {
		return CheckResult{}, fmt.Errorf("compare update versions: %w", err)
	}

	result := CheckResult{
		CurrentVersion:  current,
		LatestVersion:   release.Version,
		UpdateAvailable: comparison > 0,
		Release:         release,
	}
	if !result.UpdateAvailable {
		return result, nil
	}

	asset, err := client.SelectAsset(release)
	if err != nil {
		return CheckResult{}, err
	}
	if strings.TrimSpace(asset.URL) == "" {
		return CheckResult{}, fmt.Errorf("update asset %q has no download URL", asset.Name)
	}
	result.Asset = asset

	if normalized, ok := normalizeSHA256(asset.SHA256); ok {
		result.Asset.SHA256 = normalized
		return result, nil
	}

	checksumName := asset.Name + ".sha256"
	for _, candidate := range release.Assets {
		if candidate.Name == checksumName && strings.TrimSpace(candidate.URL) != "" {
			result.ChecksumAsset = candidate
			return result, nil
		}
	}
	return CheckResult{}, fmt.Errorf("%w for %s", ErrChecksumMissing, asset.Name)
}

func (client *Client) Download(ctx context.Context, check CheckResult, directory string, progress func(Progress)) (DownloadResult, error) {
	if !check.UpdateAvailable {
		return DownloadResult{}, errors.New("no update is available")
	}
	if strings.TrimSpace(check.Asset.Name) == "" || strings.TrimSpace(check.Asset.URL) == "" {
		return DownloadResult{}, errors.New("update asset is incomplete")
	}
	if filepath.Base(check.Asset.Name) != check.Asset.Name || check.Asset.Name == "." || check.Asset.Name == ".." {
		return DownloadResult{}, fmt.Errorf("unsafe update asset name %q", check.Asset.Name)
	}
	if strings.TrimSpace(directory) == "" {
		return DownloadResult{}, errors.New("download directory is required")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return DownloadResult{}, fmt.Errorf("create download directory: %w", err)
	}

	expected, ok := normalizeSHA256(check.Asset.SHA256)
	if !ok {
		if strings.TrimSpace(check.ChecksumAsset.URL) == "" {
			return DownloadResult{}, fmt.Errorf("%w for %s", ErrChecksumMissing, check.Asset.Name)
		}
		var err error
		expected, err = client.downloadChecksum(ctx, check.ChecksumAsset)
		if err != nil {
			return DownloadResult{}, err
		}
	}

	finalPath := filepath.Join(directory, check.Asset.Name)
	if _, err := os.Stat(finalPath); err == nil {
		return DownloadResult{}, fmt.Errorf("download destination already exists: %s", finalPath)
	} else if !os.IsNotExist(err) {
		return DownloadResult{}, fmt.Errorf("stat download destination: %w", err)
	}

	temp, err := os.CreateTemp(directory, "."+check.Asset.Name+".*.part")
	if err != nil {
		return DownloadResult{}, fmt.Errorf("create temporary update file: %w", err)
	}
	tempPath := temp.Name()
	cleanup := true
	defer func() {
		_ = temp.Close()
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	response, err := client.get(ctx, check.Asset.URL)
	if err != nil {
		return DownloadResult{}, fmt.Errorf("download update asset: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return DownloadResult{}, fmt.Errorf("download update asset: HTTP %s", response.Status)
	}

	total := check.Asset.Size
	if total <= 0 && response.ContentLength > 0 {
		total = response.ContentLength
	}
	if progress != nil {
		progress(Progress{Total: total})
	}

	hash := sha256.New()
	buffer := make([]byte, 64*1024)
	var downloaded int64
	for {
		read, readErr := response.Body.Read(buffer)
		if read > 0 {
			chunk := buffer[:read]
			if _, err := temp.Write(chunk); err != nil {
				return DownloadResult{}, fmt.Errorf("write update asset: %w", err)
			}
			if _, err := hash.Write(chunk); err != nil {
				return DownloadResult{}, fmt.Errorf("hash update asset: %w", err)
			}
			downloaded += int64(read)
			if progress != nil {
				progress(Progress{Downloaded: downloaded, Total: total})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return DownloadResult{}, fmt.Errorf("read update asset: %w", readErr)
		}
	}

	if check.Asset.Size > 0 && downloaded != check.Asset.Size {
		return DownloadResult{}, fmt.Errorf("update asset size mismatch: got %d want %d", downloaded, check.Asset.Size)
	}
	actual := hex.EncodeToString(hash.Sum(nil))
	if actual != expected {
		return DownloadResult{}, fmt.Errorf("update checksum mismatch: got %s want %s", actual, expected)
	}
	if err := temp.Sync(); err != nil {
		return DownloadResult{}, fmt.Errorf("sync update asset: %w", err)
	}
	if err := temp.Close(); err != nil {
		return DownloadResult{}, fmt.Errorf("close update asset: %w", err)
	}
	if _, err := os.Stat(finalPath); err == nil {
		return DownloadResult{}, fmt.Errorf("download destination appeared during download: %s", finalPath)
	} else if !os.IsNotExist(err) {
		return DownloadResult{}, fmt.Errorf("recheck download destination: %w", err)
	}
	if err := os.Rename(tempPath, finalPath); err != nil {
		return DownloadResult{}, fmt.Errorf("publish verified update: %w", err)
	}
	cleanup = false
	return DownloadResult{
		Asset:  check.Asset,
		Path:   finalPath,
		SHA256: actual,
		Bytes:  downloaded,
	}, nil
}

func (client *Client) downloadChecksum(ctx context.Context, asset Asset) (string, error) {
	response, err := client.get(ctx, asset.URL)
	if err != nil {
		return "", fmt.Errorf("download update checksum: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("download update checksum: HTTP %s", response.Status)
	}

	const maxChecksumBytes = 64 * 1024
	reader := io.LimitReader(response.Body, maxChecksumBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("read update checksum: %w", err)
	}
	if len(data) > maxChecksumBytes {
		return "", errors.New("update checksum file is too large")
	}
	hash, ok := checksumFromText(string(data))
	if !ok {
		return "", errors.New("update checksum file does not contain a SHA256 value")
	}
	return hash, nil
}

func (client *Client) get(ctx context.Context, rawURL string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "wails-desktop-kit-updater")
	httpClient := client.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return httpClient.Do(request)
}

func checksumFromText(value string) (string, bool) {
	scanner := bufio.NewScanner(strings.NewReader(value))
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		token := strings.TrimSpace(scanner.Text())
		token = strings.TrimPrefix(token, "sha256:")
		token = strings.Trim(token, "*()")
		if normalized, ok := normalizeSHA256(token); ok {
			return normalized, true
		}
	}
	return "", false
}

func normalizeSHA256(value string) (string, bool) {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "sha256:")
	if len(value) != sha256.Size*2 {
		return "", false
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", false
	}
	return value, true
}
