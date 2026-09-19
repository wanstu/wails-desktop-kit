package theme

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHandlerServesManifestAndStylesheet(t *testing.T) {
	server, _, wantCSS := testThemeServer(t)
	defer server.Close()
	manager, err := New(Config{
		Enabled:         true,
		ManifestURL:     server.URL + "/manifest.json",
		CacheDir:        t.TempDir(),
		RefreshInterval: time.Hour,
		HTTPClient:      server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := manager.Handler()
	manifestRec := httptest.NewRecorder()
	handler.ServeHTTP(manifestRec, httptest.NewRequest(http.MethodGet, "/desktopkit-theme/manifest.json?refresh=1", nil))
	if manifestRec.Code != http.StatusOK {
		t.Fatalf("manifest status = %d body=%s", manifestRec.Code, manifestRec.Body.String())
	}

	cssRec := httptest.NewRecorder()
	handler.ServeHTTP(cssRec, httptest.NewRequest(http.MethodGet, "/desktopkit-theme/aurora.css", nil))
	if cssRec.Code != http.StatusOK || cssRec.Body.String() != string(wantCSS) {
		t.Fatalf("css status = %d body=%q", cssRec.Code, cssRec.Body.String())
	}
	if cssRec.Header().Get("ETag") == "" {
		t.Fatal("missing stylesheet ETag")
	}
}

func TestHandlerRejectsTraversalAndMethods(t *testing.T) {
	manager, err := New(Config{Enabled: true, CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	handler := manager.Handler()

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/desktopkit-theme/a/b.css", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("traversal-like path status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/desktopkit-theme/manifest.json", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("post status = %d", rec.Code)
	}
}
