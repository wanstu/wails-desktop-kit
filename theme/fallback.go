package theme

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io/fs"
)

//go:embed fallback/*.css
var fallbackAssets embed.FS

var fallbackPackMeta = []Pack{
	{Name: "aurora", DisplayName: "极光", Description: "蓝紫冷色，适合作为通用默认主题", File: "aurora.css"},
	{Name: "ocean", DisplayName: "海洋", Description: "蓝青色调，清爽明亮", File: "ocean.css"},
	{Name: "forest", DisplayName: "森林", Description: "绿色系，柔和自然", File: "forest.css"},
	{Name: "sunset", DisplayName: "落日", Description: "橙粉暖色，更有生活感", File: "sunset.css"},
}

func fallbackManifest() Manifest {
	packs := make([]Pack, 0, len(fallbackPackMeta))
	for _, meta := range fallbackPackMeta {
		data, err := fs.ReadFile(fallbackAssets, "fallback/"+meta.File)
		if err != nil {
			continue
		}
		sum := sha256.Sum256(data)
		meta.SHA256 = hex.EncodeToString(sum[:])
		packs = append(packs, meta)
	}
	return Manifest{
		SchemaVersion: 1,
		Revision:      "builtin-fallback",
		BaseURL:       "builtin://desktopkit-theme/",
		Packs:         packs,
	}
}

func fallbackStylesheet(name string) (Pack, []byte, bool) {
	manifest := fallbackManifest()
	pack, ok := lookupPack(manifest, name)
	if !ok {
		return Pack{}, nil, false
	}
	data, err := fs.ReadFile(fallbackAssets, "fallback/"+pack.File)
	if err != nil {
		return Pack{}, nil, false
	}
	return pack, data, true
}
