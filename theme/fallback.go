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
	{Name: "aurora", DisplayName: "极光", Description: "蓝紫冷色，适合作为通用默认主题", File: "aurora.css", GradientLight: "#6366f1", GradientDark: "#818cf8", GradientLightSecondary: "#15A6A0", GradientDarkSecondary: "#B987F5", Signature: true},
	{Name: "ocean", DisplayName: "海洋", Description: "蓝青色调，清爽明亮", File: "ocean.css", GradientLight: "#0284c7", GradientDark: "#38bdf8", GradientLightSecondary: "#22B9C6", GradientDarkSecondary: "#5EE1C4", Signature: true},
	{Name: "forest", DisplayName: "森林", Description: "绿色系，柔和自然", File: "forest.css", GradientLight: "#168a57", GradientDark: "#4ade80", GradientLightSecondary: "#A1B957", GradientDarkSecondary: "#D4B577", Signature: true},
	{Name: "sunset", DisplayName: "落日", Description: "橙粉暖色，更有生活感", File: "sunset.css", GradientLight: "#e85d3f", GradientDark: "#fb7185", GradientLightSecondary: "#DF7BA3", GradientDarkSecondary: "#F49DD2", Signature: true},
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
