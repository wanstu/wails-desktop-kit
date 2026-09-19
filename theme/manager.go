package theme

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	kitpaths "github.com/wanstu/wails-desktop-kit/paths"
)

const (
	DefaultManifestURL = "https://raw.githubusercontent.com/wanstu/wails-desktop-kit-theme/master/manifest.json"
	defaultRefresh     = 6 * time.Hour
	maxManifestBytes   = 1 << 20
	maxStylesheetBytes = 512 << 10
)

var packNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

type Config struct {
	Enabled         bool
	ManifestURL     string
	CacheDir        string
	RefreshInterval time.Duration
	HTTPClient      *http.Client
}

func DefaultConfig() Config {
	return Config{
		Enabled:         true,
		ManifestURL:     DefaultManifestURL,
		RefreshInterval: defaultRefresh,
	}
}

type Pack struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	File        string `json:"file"`
	SHA256      string `json:"sha256"`
}

type Manifest struct {
	SchemaVersion int    `json:"schema_version"`
	Revision      string `json:"revision,omitempty"`
	BaseURL       string `json:"base_url"`
	Packs         []Pack `json:"packs"`
}

type Manager struct {
	cfg Config

	mu             sync.RWMutex
	manifest       Manifest
	hasManifest    bool
	manifestAt     time.Time
	manifestSource string
	lastError      string
	cacheEnabled   bool

	refreshMu sync.Mutex
}

func New(cfg Config) (*Manager, error) {
	if strings.TrimSpace(cfg.ManifestURL) == "" {
		cfg.ManifestURL = DefaultManifestURL
	}
	if cfg.RefreshInterval <= 0 {
		cfg.RefreshInterval = defaultRefresh
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 10 * time.Second}
	}
	m := &Manager{
		cfg:            cfg,
		manifest:       fallbackManifest(),
		hasManifest:    true,
		manifestSource: "builtin",
	}
	if strings.TrimSpace(cfg.CacheDir) == "" {
		root, err := kitpaths.EnsureCacheDir("wails-desktop-kit")
		if err != nil {
			m.lastError = fmt.Sprintf("theme: resolve cache dir: %v", err)
			return m, nil
		}
		cfg.CacheDir = filepath.Join(root, "themes")
		m.cfg.CacheDir = cfg.CacheDir
	}
	if err := os.MkdirAll(filepath.Join(cfg.CacheDir, "packs"), 0o700); err != nil {
		m.lastError = fmt.Sprintf("theme: create cache dir: %v", err)
		return m, nil
	}
	m.cacheEnabled = true
	if err := m.loadCachedManifest(); err != nil && !errors.Is(err, os.ErrNotExist) {
		m.lastError = err.Error()
	}
	return m, nil
}

func (m *Manager) Manifest() (Manifest, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneManifest(m.manifest), m.hasManifest
}

func (m *Manager) LastError() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastError
}

func (m *Manager) Source() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.manifestSource
}

func (m *Manager) Stale() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if !m.hasManifest {
		return true
	}
	return time.Since(m.manifestAt) >= m.cfg.RefreshInterval
}

func (m *Manager) Refresh(ctx context.Context) (Manifest, error) {
	m.refreshMu.Lock()
	defer m.refreshMu.Unlock()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, m.cfg.ManifestURL, nil)
	if err != nil {
		return Manifest{}, fmt.Errorf("theme: create manifest request: %w", err)
	}
	response, err := m.cfg.HTTPClient.Do(request)
	if err != nil {
		m.setLastError(err)
		return Manifest{}, fmt.Errorf("theme: fetch manifest: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err := fmt.Errorf("theme: fetch manifest: HTTP %d", response.StatusCode)
		m.setLastError(err)
		return Manifest{}, err
	}
	data, err := readLimited(response.Body, maxManifestBytes)
	if err != nil {
		m.setLastError(err)
		return Manifest{}, fmt.Errorf("theme: read manifest: %w", err)
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		m.setLastError(err)
		return Manifest{}, fmt.Errorf("theme: decode manifest: %w", err)
	}
	if err := validateManifest(manifest); err != nil {
		m.setLastError(err)
		return Manifest{}, err
	}
	if m.cacheEnabled {
		if err := m.syncManifestStylesheets(ctx, manifest); err != nil {
			m.setLastError(err)
			return Manifest{}, err
		}
	}
	var cacheErr error
	if m.cacheEnabled {
		normalized, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			return Manifest{}, fmt.Errorf("theme: encode manifest cache: %w", err)
		}
		normalized = append(normalized, '\n')
		if err := atomicWrite(filepath.Join(m.cfg.CacheDir, "manifest.json"), normalized, 0o600); err != nil {
			cacheErr = fmt.Errorf("theme: save manifest cache: %w", err)
		}
	}

	now := time.Now()
	m.mu.Lock()
	m.manifest = manifest
	m.hasManifest = true
	m.manifestAt = now
	m.manifestSource = "remote"
	if cacheErr != nil {
		m.lastError = cacheErr.Error()
	} else {
		m.lastError = ""
	}
	m.mu.Unlock()
	return cloneManifest(manifest), nil
}

func (m *Manager) RefreshInBackgroundIfStale() {
	if !m.Stale() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		_, _ = m.Refresh(ctx)
	}()
}

func (m *Manager) EnsureStylesheet(ctx context.Context, name string) (Pack, []byte, error) {
	if !packNamePattern.MatchString(name) {
		return Pack{}, nil, fmt.Errorf("theme: invalid pack name %q", name)
	}

	manifest, ok := m.Manifest()
	if !ok {
		var err error
		manifest, err = m.Refresh(ctx)
		if err != nil {
			return Pack{}, nil, err
		}
	}
	pack, ok := lookupPack(manifest, name)
	if !ok {
		refreshed, err := m.Refresh(ctx)
		if err == nil {
			manifest = refreshed
			pack, ok = lookupPack(manifest, name)
		}
	}
	if !ok {
		if fallbackPack, fallbackCSS, fallbackOK := fallbackStylesheet(name); fallbackOK {
			return fallbackPack, fallbackCSS, nil
		}
		return Pack{}, nil, fmt.Errorf("theme: pack %q not found", name)
	}

	if m.cacheEnabled {
		cachePath := m.packCachePath(pack)
		if data, err := os.ReadFile(cachePath); err == nil {
			if stylesheetValid(pack, data) == nil {
				return pack, data, nil
			}
			_ = os.Remove(cachePath)
		}
	}

	if fallbackPack, fallbackCSS, fallbackOK := fallbackStylesheet(name); fallbackOK &&
		(manifest.Revision == "builtin-fallback" || strings.EqualFold(pack.SHA256, fallbackPack.SHA256)) {
		return fallbackPack, fallbackCSS, nil
	}

	base, err := url.Parse(manifest.BaseURL)
	if err != nil {
		return Pack{}, nil, fmt.Errorf("theme: parse base URL: %w", err)
	}
	assetURL, err := base.Parse(pack.File)
	if err != nil {
		return Pack{}, nil, fmt.Errorf("theme: resolve stylesheet URL: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL.String(), nil)
	if err != nil {
		return Pack{}, nil, fmt.Errorf("theme: create stylesheet request: %w", err)
	}
	response, err := m.cfg.HTTPClient.Do(request)
	if err != nil {
		return fallbackOrError(name, fmt.Errorf("theme: fetch %s: %w", name, err))
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fallbackOrError(name, fmt.Errorf("theme: fetch %s: HTTP %d", name, response.StatusCode))
	}
	data, err := readLimited(response.Body, maxStylesheetBytes)
	if err != nil {
		return fallbackOrError(name, fmt.Errorf("theme: read %s: %w", name, err))
	}
	if err := stylesheetValid(pack, data); err != nil {
		return fallbackOrError(name, err)
	}
	if m.cacheEnabled {
		if err := atomicWrite(m.packCachePath(pack), data, 0o600); err != nil {
			m.setLastError(fmt.Errorf("theme: cache %s: %w", name, err))
		}
	}
	return pack, data, nil
}

func (m *Manager) syncManifestStylesheets(ctx context.Context, manifest Manifest) error {
	if !m.cacheEnabled {
		return nil
	}
	base, err := url.Parse(manifest.BaseURL)
	if err != nil {
		return fmt.Errorf("theme: parse snapshot base URL: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	workerCount := 6
	if len(manifest.Packs) < workerCount {
		workerCount = len(manifest.Packs)
	}

	jobs := make(chan Pack)
	var wg sync.WaitGroup
	var errMu sync.Mutex
	var firstErr error

	worker := func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case pack, ok := <-jobs:
				if !ok {
					return
				}
				if err := m.syncManifestPack(ctx, base, pack); err != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = err
						cancel()
					}
					errMu.Unlock()
					return
				}
			}
		}
	}

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}

feedLoop:
	for _, pack := range manifest.Packs {
		select {
		case <-ctx.Done():
			break feedLoop
		case jobs <- pack:
		}
	}
	close(jobs)
	wg.Wait()

	errMu.Lock()
	defer errMu.Unlock()
	if firstErr != nil {
		return fmt.Errorf("theme: sync snapshot: %w", firstErr)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("theme: sync snapshot: %w", err)
	}
	return nil
}

func (m *Manager) syncManifestPack(ctx context.Context, base *url.URL, pack Pack) error {
	cachePath := m.packCachePath(pack)
	if data, err := os.ReadFile(cachePath); err == nil && stylesheetValid(pack, data) == nil {
		return nil
	}

	assetURL, err := base.Parse(pack.File)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", pack.Name, err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create %s request: %w", pack.Name, err)
	}
	response, err := m.cfg.HTTPClient.Do(request)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", pack.Name, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch %s: HTTP %d", pack.Name, response.StatusCode)
	}
	data, err := readLimited(response.Body, maxStylesheetBytes)
	if err != nil {
		return fmt.Errorf("read %s: %w", pack.Name, err)
	}
	if err := stylesheetValid(pack, data); err != nil {
		return err
	}
	if err := atomicWrite(cachePath, data, 0o600); err != nil {
		return fmt.Errorf("cache %s: %w", pack.Name, err)
	}
	return nil
}

func (m *Manager) validateCachedSnapshot(manifest Manifest) error {
	for _, pack := range manifest.Packs {
		data, err := os.ReadFile(m.packCachePath(pack))
		if err != nil {
			return fmt.Errorf("%s: %w", pack.Name, err)
		}
		if err := stylesheetValid(pack, data); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) loadCachedManifest() error {
	path := filepath.Join(m.cfg.CacheDir, "manifest.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return fmt.Errorf("theme: decode cached manifest: %w", err)
	}
	if err := validateManifest(manifest); err != nil {
		return fmt.Errorf("theme: validate cached manifest: %w", err)
	}
	if err := m.validateCachedSnapshot(manifest); err != nil {
		return fmt.Errorf("theme: cached snapshot incomplete: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.manifest = manifest
	m.hasManifest = true
	m.manifestAt = info.ModTime()
	m.manifestSource = "cache"
	m.mu.Unlock()
	return nil
}

func (m *Manager) packCachePath(pack Pack) string {
	return filepath.Join(m.cfg.CacheDir, "packs", pack.Name+"-"+pack.SHA256+".css")
}

func (m *Manager) setLastError(err error) {
	if err == nil {
		return
	}
	m.mu.Lock()
	m.lastError = err.Error()
	m.mu.Unlock()
}

func fallbackOrError(name string, err error) (Pack, []byte, error) {
	if pack, data, ok := fallbackStylesheet(name); ok {
		return pack, data, nil
	}
	return Pack{}, nil, err
}

func lookupPack(manifest Manifest, name string) (Pack, bool) {
	for _, pack := range manifest.Packs {
		if pack.Name == name {
			return pack, true
		}
	}
	return Pack{}, false
}

func validateManifest(manifest Manifest) error {
	if manifest.SchemaVersion != 1 {
		return fmt.Errorf("theme: unsupported manifest schema %d", manifest.SchemaVersion)
	}
	base, err := url.Parse(manifest.BaseURL)
	if err != nil || (base.Scheme != "https" && base.Scheme != "http") || base.Host == "" || base.User != nil {
		return fmt.Errorf("theme: invalid manifest base URL %q", manifest.BaseURL)
	}
	if len(manifest.Packs) == 0 || len(manifest.Packs) > 256 {
		return fmt.Errorf("theme: manifest pack count %d is invalid", len(manifest.Packs))
	}
	seen := make(map[string]struct{}, len(manifest.Packs))
	for _, pack := range manifest.Packs {
		if !packNamePattern.MatchString(pack.Name) {
			return fmt.Errorf("theme: invalid pack name %q", pack.Name)
		}
		if _, ok := seen[pack.Name]; ok {
			return fmt.Errorf("theme: duplicate pack %q", pack.Name)
		}
		seen[pack.Name] = struct{}{}
		if strings.TrimSpace(pack.DisplayName) == "" || strings.TrimSpace(pack.Description) == "" {
			return fmt.Errorf("theme: incomplete metadata for %q", pack.Name)
		}
		if pack.File != pack.Name+".css" {
			return fmt.Errorf("theme: pack %q file %q is invalid", pack.Name, pack.File)
		}
		if len(pack.SHA256) != sha256.Size*2 {
			return fmt.Errorf("theme: pack %q has invalid sha256", pack.Name)
		}
		if _, err := hex.DecodeString(pack.SHA256); err != nil {
			return fmt.Errorf("theme: pack %q has invalid sha256: %w", pack.Name, err)
		}
	}
	return nil
}

func stylesheetValid(pack Pack, data []byte) error {
	sum := sha256.Sum256(data)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), pack.SHA256) {
		return fmt.Errorf("theme: stylesheet checksum mismatch for %q", pack.Name)
	}
	if !utf8.Valid(data) {
		return fmt.Errorf("theme: stylesheet %q is not UTF-8", pack.Name)
	}
	css := strings.ToLower(string(data))
	if strings.Contains(css, "@import") || strings.Contains(css, "url(") {
		return fmt.Errorf("theme: stylesheet %q contains external resource directives", pack.Name)
	}
	if !strings.Contains(string(data), `data-dk-theme-pack="`+pack.Name+`"`) {
		return fmt.Errorf("theme: stylesheet %q is missing its pack selector", pack.Name)
	}
	return nil
}

func readLimited(reader io.Reader, limit int64) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("response exceeds %d bytes", limit)
	}
	return data, nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(tmpPath, path)
}

func cloneManifest(manifest Manifest) Manifest {
	result := manifest
	result.Packs = append([]Pack(nil), manifest.Packs...)
	return result
}
