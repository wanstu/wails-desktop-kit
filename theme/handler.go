package theme

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const assetPrefix = "/desktopkit-theme/"

type catalogResponse struct {
	SchemaVersion int    `json:"schema_version"`
	Revision      string `json:"revision,omitempty"`
	Source        string `json:"source"`
	Stale         bool   `json:"stale"`
	LastError     string `json:"last_error,omitempty"`
	Packs         []Pack `json:"packs"`
}

func (m *Manager) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, assetPrefix) {
			if r.Method == http.MethodGet {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		relative := strings.TrimPrefix(r.URL.Path, assetPrefix)
		if relative == "manifest.json" {
			m.serveManifest(w, r)
			return
		}
		if strings.Contains(relative, "/") || !strings.HasSuffix(relative, ".css") {
			http.NotFound(w, r)
			return
		}
		name := strings.TrimSuffix(relative, ".css")
		m.serveStylesheet(w, r, name)
	})
}

func (m *Manager) serveManifest(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("refresh") == "1"
	source := m.Source()
	var manifest Manifest
	var ok bool
	var refreshErr error

	if force {
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		manifest, refreshErr = m.Refresh(ctx)
		if refreshErr == nil {
			ok = true
			source = "remote"
		}
	}
	if !ok {
		manifest, ok = m.Manifest()
	}
	if !ok {
		ctx, cancel := context.WithTimeout(r.Context(), 12*time.Second)
		defer cancel()
		manifest, refreshErr = m.Refresh(ctx)
		if refreshErr == nil {
			ok = true
			source = "remote"
		}
	}
	if !ok {
		http.Error(w, fmt.Sprintf("theme catalog unavailable: %v", refreshErr), http.StatusServiceUnavailable)
		return
	}
	if !force {
		m.RefreshInBackgroundIfStale()
	}

	response := catalogResponse{
		SchemaVersion: manifest.SchemaVersion,
		Revision:      manifest.Revision,
		Source:        source,
		Stale:         m.Stale(),
		LastError:     m.LastError(),
		Packs:         manifest.Packs,
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

func (m *Manager) serveStylesheet(w http.ResponseWriter, r *http.Request, name string) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	pack, data, err := m.EnsureStylesheet(ctx, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	etag := `"` + pack.SHA256 + `"`
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("ETag", etag)
	_, _ = w.Write(data)
}
