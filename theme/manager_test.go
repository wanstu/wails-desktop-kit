package theme

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testThemeServer(t *testing.T) (*httptest.Server, Manifest, []byte) {
	t.Helper()
	css := []byte(`:root[data-dk-theme-pack="aurora"] { --dk-primary: #123456; }
:root[data-dk-theme="dark"][data-dk-theme-pack="aurora"] { --dk-primary: #abcdef; }
`)
	sum := sha256.Sum256(css)
	var manifest Manifest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(manifest)
		case "/assets/aurora.css":
			w.Header().Set("Content-Type", "text/css")
			_, _ = w.Write(css)
		default:
			http.NotFound(w, r)
		}
	}))
	manifest = Manifest{
		SchemaVersion: 1,
		Revision:      "test-1",
		BaseURL:       server.URL + "/assets/",
		Packs: []Pack{{
			Name:        "aurora",
			DisplayName: "极光",
			Description: "测试主题",
			File:        "aurora.css",
			SHA256:      hex.EncodeToString(sum[:]),
		}},
	}
	return server, manifest, css
}

func TestManagerStartsWithFourBuiltInFallbackThemes(t *testing.T) {
	manager, err := New(Config{
		Enabled:     true,
		ManifestURL: "https://example.invalid/manifest.json",
		CacheDir:    t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, ok := manager.Manifest()
	if !ok {
		t.Fatal("expected built-in fallback manifest")
	}
	if manager.Source() != "builtin" || manifest.Revision != "builtin-fallback" || len(manifest.Packs) != 4 {
		t.Fatalf("fallback manifest = %+v source=%q", manifest, manager.Source())
	}
	for _, name := range []string{"aurora", "ocean", "forest", "sunset"} {
		pack, css, err := manager.EnsureStylesheet(context.Background(), name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if pack.Name != name || len(css) == 0 {
			t.Fatalf("%s fallback invalid: pack=%+v bytes=%d", name, pack, len(css))
		}
	}
}

func TestManagerRefreshAndCacheStylesheet(t *testing.T) {
	server, wantManifest, wantCSS := testThemeServer(t)
	defer server.Close()

	cacheDir := t.TempDir()
	manager, err := New(Config{
		Enabled:         true,
		ManifestURL:     server.URL + "/manifest.json",
		CacheDir:        cacheDir,
		RefreshInterval: time.Hour,
		HTTPClient:      server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	gotManifest, err := manager.Refresh(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if gotManifest.Revision != wantManifest.Revision || len(gotManifest.Packs) != 1 {
		t.Fatalf("manifest = %+v", gotManifest)
	}

	pack, gotCSS, err := manager.EnsureStylesheet(context.Background(), "aurora")
	if err != nil {
		t.Fatal(err)
	}
	if pack.SHA256 != wantManifest.Packs[0].SHA256 || string(gotCSS) != string(wantCSS) {
		t.Fatalf("unexpected stylesheet result")
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "manifest.json")); err != nil {
		t.Fatalf("manifest cache missing: %v", err)
	}
	if _, err := os.Stat(manager.packCachePath(pack)); err != nil {
		t.Fatalf("stylesheet cache missing: %v", err)
	}
}

func TestManagerUsesCachedManifestOffline(t *testing.T) {
	server, _, _ := testThemeServer(t)
	cacheDir := t.TempDir()
	manager, err := New(Config{
		Enabled:         true,
		ManifestURL:     server.URL + "/manifest.json",
		CacheDir:        cacheDir,
		RefreshInterval: time.Hour,
		HTTPClient:      server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	server.Close()

	offline, err := New(Config{
		Enabled:         true,
		ManifestURL:     server.URL + "/manifest.json",
		CacheDir:        cacheDir,
		RefreshInterval: time.Hour,
		HTTPClient:      &http.Client{Timeout: 100 * time.Millisecond},
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest, ok := offline.Manifest()
	if !ok || len(manifest.Packs) != 1 || manifest.Packs[0].Name != "aurora" {
		t.Fatalf("cached manifest unavailable: %+v ok=%v", manifest, ok)
	}
}

func TestRefreshCachesCompleteSnapshotForOfflineUse(t *testing.T) {
	styles := map[string][]byte{
		"aurora": []byte(`:root[data-dk-theme-pack="aurora"] { --dk-primary: #123456; }`),
		"ocean":  []byte(`:root[data-dk-theme-pack="ocean"] { --dk-primary: #0284c7; }`),
	}
	manifest := Manifest{
		SchemaVersion: 1,
		Revision:      "snapshot-1",
		Packs:         make([]Pack, 0, len(styles)),
	}
	for _, name := range []string{"aurora", "ocean"} {
		sum := sha256.Sum256(styles[name])
		manifest.Packs = append(manifest.Packs, Pack{
			Name: name, DisplayName: name, Description: name + " theme",
			File: name + ".css", SHA256: hex.EncodeToString(sum[:]),
		})
	}

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			manifest.BaseURL = server.URL + "/assets/"
			_ = json.NewEncoder(w).Encode(manifest)
		case "/assets/aurora.css":
			_, _ = w.Write(styles["aurora"])
		case "/assets/ocean.css":
			_, _ = w.Write(styles["ocean"])
		default:
			http.NotFound(w, r)
		}
	}))

	cacheDir := t.TempDir()
	manager, err := New(Config{
		Enabled: true, ManifestURL: server.URL + "/manifest.json",
		CacheDir: cacheDir, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	server.Close()

	offline, err := New(Config{
		Enabled: true, ManifestURL: "http://127.0.0.1:1/manifest.json",
		CacheDir: cacheDir, HTTPClient: &http.Client{Timeout: 50 * time.Millisecond},
	})
	if err != nil {
		t.Fatal(err)
	}
	cached, ok := offline.Manifest()
	if !ok || offline.Source() != "cache" || len(cached.Packs) != 2 {
		t.Fatalf("offline snapshot = %+v source=%q ok=%v", cached, offline.Source(), ok)
	}
	for _, name := range []string{"aurora", "ocean"} {
		pack, css, err := offline.EnsureStylesheet(context.Background(), name)
		if err != nil {
			t.Fatalf("%s offline: %v", name, err)
		}
		if pack.Name != name || string(css) != string(styles[name]) {
			t.Fatalf("%s offline mismatch", name)
		}
	}
}

func TestRefreshKeepsLastKnownGoodSnapshotOnPartialFailure(t *testing.T) {
	oldCSS := []byte(`:root[data-dk-theme-pack="aurora"] { --dk-primary: #111111; }`)
	oldSum := sha256.Sum256(oldCSS)
	oldManifest := Manifest{
		SchemaVersion: 1,
		Revision:      "good",
		Packs: []Pack{{
			Name: "aurora", DisplayName: "极光", Description: "good",
			File: "aurora.css", SHA256: hex.EncodeToString(oldSum[:]),
		}},
	}

	mode := "good"
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/manifest.json" {
			if mode == "good" {
				oldManifest.BaseURL = server.URL + "/assets/"
				_ = json.NewEncoder(w).Encode(oldManifest)
				return
			}
			badCSS := []byte(`:root[data-dk-theme-pack="future"] { --dk-primary: #222222; }`)
			badSum := sha256.Sum256(badCSS)
			_ = json.NewEncoder(w).Encode(Manifest{
				SchemaVersion: 1,
				Revision:      "partial-bad",
				BaseURL:       server.URL + "/assets/",
				Packs: []Pack{
					{Name: "aurora", DisplayName: "极光", Description: "good", File: "aurora.css", SHA256: hex.EncodeToString(oldSum[:])},
					{Name: "future", DisplayName: "未来", Description: "missing", File: "future.css", SHA256: hex.EncodeToString(badSum[:])},
				},
			})
			return
		}
		if r.URL.Path == "/assets/aurora.css" {
			_, _ = w.Write(oldCSS)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	manager, err := New(Config{
		Enabled: true, ManifestURL: server.URL + "/manifest.json",
		CacheDir: cacheDir, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	mode = "bad"
	if _, err := manager.Refresh(context.Background()); err == nil {
		t.Fatal("expected partial snapshot refresh to fail")
	}

	current, ok := manager.Manifest()
	if !ok || current.Revision != "good" || len(current.Packs) != 1 {
		t.Fatalf("last known good was replaced: %+v", current)
	}

	reloaded, err := New(Config{
		Enabled: true, ManifestURL: server.URL + "/manifest.json",
		CacheDir: cacheDir, HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	persisted, ok := reloaded.Manifest()
	if !ok || reloaded.Source() != "cache" || persisted.Revision != "good" {
		t.Fatalf("persisted snapshot changed: %+v source=%q", persisted, reloaded.Source())
	}
}

func TestValidateManifestRejectsUnsafePack(t *testing.T) {
	manifest := Manifest{
		SchemaVersion: 1,
		BaseURL:       "https://example.com/assets/",
		Packs: []Pack{{
			Name:        "../evil",
			DisplayName: "bad",
			Description: "bad",
			File:        "../evil.css",
			SHA256:      string(make([]byte, 64)),
		}},
	}
	if err := validateManifest(manifest); err == nil {
		t.Fatal("expected unsafe pack to be rejected")
	}
}

func TestStylesheetRejectsExternalResources(t *testing.T) {
	data := []byte(`:root[data-dk-theme-pack="aurora"] { --dk-bg-app: url(https://example.com/x); }`)
	sum := sha256.Sum256(data)
	pack := Pack{Name: "aurora", SHA256: hex.EncodeToString(sum[:])}
	if err := stylesheetValid(pack, data); err == nil {
		t.Fatal("expected external resource directive to be rejected")
	}
}
