package updater

import (
	"errors"
	"testing"
)

func TestAssetSelectors(t *testing.T) {
	release := Release{Assets: []Asset{
		{Name: "demo-v1.2.3-windows-amd64.exe"},
		{Name: "demo-v1.2.3-windows-amd64-setup.exe"},
		{Name: "demo-v1.2.3-windows-amd64-setup.exe.sha256"},
	}}

	asset, err := ExactAsset("demo-v1.2.3-windows-amd64.exe")(release)
	if err != nil || asset.Name != "demo-v1.2.3-windows-amd64.exe" {
		t.Fatalf("ExactAsset: asset=%+v err=%v", asset, err)
	}

	asset, err = AssetBySuffix("windows-amd64-setup.exe")(release)
	if err != nil || asset.Name != "demo-v1.2.3-windows-amd64-setup.exe" {
		t.Fatalf("AssetBySuffix: asset=%+v err=%v", asset, err)
	}
}

func TestAssetBySuffixRejectsAmbiguousMatch(t *testing.T) {
	release := Release{Assets: []Asset{
		{Name: "one-windows-amd64-setup.exe"},
		{Name: "two-windows-amd64-setup.exe"},
	}}
	if _, err := AssetBySuffix("windows-amd64-setup.exe")(release); err == nil {
		t.Fatal("expected ambiguous selector error")
	}
}

func TestExactAssetNotFound(t *testing.T) {
	_, err := ExactAsset("missing")(Release{})
	if !errors.Is(err, ErrAssetNotFound) {
		t.Fatalf("expected ErrAssetNotFound, got %v", err)
	}
}
