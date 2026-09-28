package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type GitHubProvider struct {
	Owner             string
	Repository        string
	Token             string
	IncludePrerelease bool
	BaseURL           string
	HTTPClient        *http.Client
}

type githubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	HTMLURL     string        `json:"html_url"`
	Draft       bool          `json:"draft"`
	Prerelease  bool          `json:"prerelease"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []githubAsset `json:"assets"`
}

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
	Digest             string `json:"digest"`
}

func (provider GitHubProvider) Latest(ctx context.Context) (Release, error) {
	owner := strings.TrimSpace(provider.Owner)
	repository := strings.TrimSpace(provider.Repository)
	if owner == "" || repository == "" || strings.Contains(owner, "/") || strings.Contains(repository, "/") {
		return Release{}, errors.New("GitHub owner and repository are required and must not contain slashes")
	}

	baseURL := strings.TrimRight(strings.TrimSpace(provider.BaseURL), "/")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	endpoint := fmt.Sprintf("%s/repos/%s/%s/releases?per_page=30", baseURL, url.PathEscape(owner), url.PathEscape(repository))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Release{}, fmt.Errorf("create GitHub release request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "wails-desktop-kit-updater")
	if token := strings.TrimSpace(provider.Token); token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	httpClient := provider.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return Release{}, fmt.Errorf("load GitHub releases: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 8*1024))
		return Release{}, fmt.Errorf("load GitHub releases: HTTP %s: %s", response.Status, strings.TrimSpace(string(body)))
	}

	const maxReleaseJSON = 4 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(response.Body, maxReleaseJSON+1))
	if err != nil {
		return Release{}, fmt.Errorf("read GitHub releases: %w", err)
	}
	if len(data) > maxReleaseJSON {
		return Release{}, errors.New("GitHub releases response is too large")
	}
	var releases []githubRelease
	if err := json.Unmarshal(data, &releases); err != nil {
		return Release{}, fmt.Errorf("decode GitHub releases: %w", err)
	}

	var best Release
	found := false
	for _, release := range releases {
		if release.Draft || (release.Prerelease && !provider.IncludePrerelease) {
			continue
		}
		if _, err := parseVersion(release.TagName); err != nil {
			continue
		}
		candidate := Release{
			Version:     release.TagName,
			Name:        release.Name,
			PageURL:     release.HTMLURL,
			Prerelease:  release.Prerelease,
			PublishedAt: release.PublishedAt,
			Assets:      make([]Asset, 0, len(release.Assets)),
		}
		for _, asset := range release.Assets {
			sha, _ := normalizeSHA256(asset.Digest)
			candidate.Assets = append(candidate.Assets, Asset{
				Name:   asset.Name,
				URL:    asset.BrowserDownloadURL,
				Size:   asset.Size,
				SHA256: sha,
			})
		}
		if !found {
			best = candidate
			found = true
			continue
		}
		comparison, err := CompareVersions(candidate.Version, best.Version)
		if err != nil {
			continue
		}
		if comparison > 0 || (comparison == 0 && candidate.PublishedAt.After(best.PublishedAt)) {
			best = candidate
		}
	}
	if !found {
		return Release{}, ErrNoRelease
	}
	return best, nil
}
