package desktopkit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	wailsassets "github.com/wailsapp/wails/v2/pkg/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	kittheme "github.com/wanstu/wails-desktop-kit/theme"
	kitui "github.com/wanstu/wails-desktop-kit/ui"
)

type themeTestLog struct{}

func (themeTestLog) Debug(string, ...interface{}) {}
func (themeTestLog) Error(string, ...interface{}) {}

func TestRuntimeThemeIsServedThroughWailsAssetFallback(t *testing.T) {
	css := []byte(`:root[data-dk-theme-pack="aurora"] { --dk-primary: #123456; }
:root[data-dk-theme="dark"][data-dk-theme-pack="aurora"] { --dk-primary: #abcdef; }
`)
	sum := sha256.Sum256(css)
	var manifest kittheme.Manifest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			_ = json.NewEncoder(w).Encode(manifest)
		case "/assets/aurora.css":
			_, _ = w.Write(css)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	manifest = kittheme.Manifest{
		SchemaVersion: 1,
		BaseURL:       server.URL + "/assets/",
		Packs: []kittheme.Pack{{
			Name: "aurora", DisplayName: "极光", Description: "测试主题",
			File: "aurora.css", SHA256: hex.EncodeToString(sum[:]),
		}},
	}

	cfg := DefaultThemeConfig()
	cfg.ManifestURL = server.URL + "/manifest.json"
	cfg.CacheDir = t.TempDir()
	cfg.HTTPClient = server.Client()

	assets := kitui.Mount(fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html>app</html>")},
	})
	handler, err := wailsassets.NewAssetHandler(assetserver.Options{
		Assets:     assets,
		Middleware: newThemeAssetMiddleware(cfg),
	}, themeTestLog{})
	if err != nil {
		t.Fatal(err)
	}

	for _, target := range []string{
		"/desktopkit/tokens.css",
		"/desktopkit-theme/manifest.json?refresh=1",
		"/desktopkit-theme/aurora.css",
	} {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
			t.Fatalf("%s: status=%d body=%s", target, rec.Code, rec.Body.String())
		}
	}
}

func TestRuntimeThemeInitFailureDoesNotAbortApplicationAssets(t *testing.T) {
	root := t.TempDir()
	blocked := filepath.Join(root, "not-a-directory")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultThemeConfig()
	cfg.CacheDir = blocked
	middleware := newThemeAssetMiddleware(cfg)
	if middleware == nil {
		t.Fatal("expected fallback middleware")
	}
	handler := middleware(http.NotFoundHandler())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/desktopkit-theme/manifest.json", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("manifest status = %d body=%s", rec.Code, rec.Body.String())
	}
	var catalog struct {
		Source string          `json:"source"`
		Packs  []kittheme.Pack `json:"packs"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	if catalog.Source != "builtin" || len(catalog.Packs) != 4 {
		t.Fatalf("fallback catalog = %+v", catalog)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/desktopkit-theme/ocean.css", nil))
	if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
		t.Fatalf("fallback stylesheet status=%d body=%s", rec.Code, rec.Body.String())
	}
}
