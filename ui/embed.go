// Package ui exposes the shared desktop-kit CSS design system.
//
// Applications may call Mount to serve the styles under /desktopkit/* while
// keeping their own embedded frontend as the primary asset filesystem.
package ui

import (
	"embed"
	"io/fs"
	"path"
	"strings"
)

//go:embed assets/*.css
var embedded embed.FS

type mountedFS struct {
	app fs.FS
}

// Mount overlays the desktop-kit stylesheet assets onto an application FS.
// The shared files are available as desktopkit/tokens.css,
// desktopkit/base.css, desktopkit/components.css, and
// desktopkit/navigation.css.
func Mount(app fs.FS) fs.FS {
	return mountedFS{app: app}
}

func (m mountedFS) Open(name string) (fs.File, error) {
	clean := path.Clean(strings.TrimPrefix(name, "/"))
	if strings.HasPrefix(clean, "desktopkit/") {
		return embedded.Open("assets/" + strings.TrimPrefix(clean, "desktopkit/"))
	}
	return m.app.Open(clean)
}
