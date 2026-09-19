package desktopkit

import (
	"net/http"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	kittheme "github.com/wanstu/wails-desktop-kit/theme"
)

// ThemeConfig configures the optional runtime Theme Pack manager.
// When enabled, theme assets are fetched and cached at runtime instead of
// being compiled into each application binary.
type ThemeConfig = kittheme.Config

// DefaultThemeConfig enables the official Desktop Kit Theme source with
// conservative refresh and cache defaults.
func DefaultThemeConfig() ThemeConfig {
	return kittheme.DefaultConfig()
}

func newThemeAssetMiddleware(cfg ThemeConfig) assetserver.Middleware {
	if !cfg.Enabled {
		return nil
	}

	manager, err := kittheme.New(cfg)
	var themeHandler http.Handler
	if err == nil {
		themeHandler = manager.Handler()
	} else {
		message := "desktop-kit theme runtime unavailable: " + err.Error()
		themeHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, message, http.StatusServiceUnavailable)
		})
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/desktopkit-theme/") {
				themeHandler.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
