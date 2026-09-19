package ui

import (
	"io/fs"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	wailsassets "github.com/wailsapp/wails/v2/pkg/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

type silentLog struct{}

func (silentLog) Debug(string, ...interface{}) {}
func (silentLog) Error(string, ...interface{}) {}
func TestMountedFilesystemContract(t *testing.T) {
	m := Mount(fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("app")}})
	if err := fstest.TestFS(m, "index.html", "desktopkit/tokens.css", "desktopkit/base.css", "desktopkit/components.css", "desktopkit/navigation.css", "desktopkit/theme.js"); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"/index.html", "../index.html", "desktopkit/../index.html", "desktopkit//tokens.css"} {
		if _, err := m.Open(name); err == nil {
			t.Fatalf("accepted invalid FS path %q", name)
		}
	}
}
func TestWailsServesMountedStylesFromExplicitFrontendRoot(t *testing.T) {
	for _, root := range []string{".", "frontend/src"} {
		t.Run(root, func(t *testing.T) {
			name := "index.html"
			if root != "." {
				name = root + "/" + name
			}
			app := fstest.MapFS{name: &fstest.MapFile{Data: []byte("<html>app</html>")}}
			sub, err := fs.Sub(app, root)
			if err != nil {
				t.Fatal(err)
			}
			handler, err := wailsassets.NewAssetHandler(assetserver.Options{Assets: Mount(sub)}, silentLog{})
			if err != nil {
				t.Fatal(err)
			}
			for _, url := range []string{"/index.html", "/desktopkit/tokens.css", "/desktopkit/components.css", "/desktopkit/theme.js"} {
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, httptest.NewRequest("GET", url, nil))
				if rec.Code != 200 || rec.Body.Len() == 0 {
					t.Fatalf("%s: status=%d bytes=%d", url, rec.Code, rec.Body.Len())
				}
			}
		})
	}
}
