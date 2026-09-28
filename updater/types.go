package updater

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrNoRelease       = errors.New("no eligible release found")
	ErrAssetNotFound   = errors.New("update asset not found")
	ErrChecksumMissing = errors.New("update checksum is missing")
)

type Asset struct {
	Name   string
	URL    string
	Size   int64
	SHA256 string
}

type Release struct {
	Version     string
	Name        string
	PageURL     string
	Prerelease  bool
	PublishedAt time.Time
	Assets      []Asset
}

type Provider interface {
	Latest(context.Context) (Release, error)
}

type AssetSelector func(Release) (Asset, error)

func ExactAsset(name string) AssetSelector {
	name = strings.TrimSpace(name)
	return func(release Release) (Asset, error) {
		for _, asset := range release.Assets {
			if asset.Name == name {
				return asset, nil
			}
		}
		return Asset{}, fmt.Errorf("%w: %s", ErrAssetNotFound, name)
	}
}

func AssetBySuffix(suffix string) AssetSelector {
	suffix = strings.TrimSpace(suffix)
	return func(release Release) (Asset, error) {
		var match Asset
		count := 0
		for _, asset := range release.Assets {
			if strings.HasSuffix(asset.Name, suffix) {
				match = asset
				count++
			}
		}
		switch count {
		case 0:
			return Asset{}, fmt.Errorf("%w: suffix %q", ErrAssetNotFound, suffix)
		case 1:
			return match, nil
		default:
			return Asset{}, fmt.Errorf("multiple update assets match suffix %q", suffix)
		}
	}
}

type CheckResult struct {
	CurrentVersion  string
	LatestVersion   string
	UpdateAvailable bool
	Release         Release
	Asset           Asset
	ChecksumAsset   Asset
}

type Progress struct {
	Downloaded int64
	Total      int64
}

type DownloadResult struct {
	Asset  Asset
	Path   string
	SHA256 string
	Bytes  int64
}
