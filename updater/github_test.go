package updater

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubProviderSelectsHighestEligibleRelease(t *testing.T) {
	var tokenSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/demo/releases" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("per_page") != "30" {
			t.Fatalf("per_page = %q", r.URL.Query().Get("per_page"))
		}
		tokenSeen = r.Header.Get("Authorization") == "Bearer test-token"
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[
		  {
		    "tag_name":"v9.9.9","name":"draft","draft":true,"prerelease":false,
		    "published_at":"2026-09-01T00:00:00Z","assets":[]
		  },
		  {
		    "tag_name":"v2.0.0-rc.1","name":"rc","draft":false,"prerelease":true,
		    "published_at":"2026-09-03T00:00:00Z",
		    "assets":[{"name":"demo-v2.0.0-rc.1.exe","browser_download_url":"https://example.test/rc","size":12,"digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]
		  },
		  {
		    "tag_name":"v1.4.0","name":"stable","draft":false,"prerelease":false,
		    "published_at":"2026-09-02T00:00:00Z",
		    "assets":[{"name":"demo-v1.4.0.exe","browser_download_url":"https://example.test/stable","size":10,"digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]
		  }
		]`)
	}))
	defer server.Close()

	stable, err := (GitHubProvider{
		Owner:      "acme",
		Repository: "demo",
		Token:      "test-token",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}).Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !tokenSeen {
		t.Fatal("expected Authorization header")
	}
	if stable.Version != "v1.4.0" || stable.Prerelease {
		t.Fatalf("stable release = %+v", stable)
	}
	if len(stable.Assets) != 1 || stable.Assets[0].SHA256 != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("stable assets = %+v", stable.Assets)
	}

	withPrerelease, err := (GitHubProvider{
		Owner:             "acme",
		Repository:        "demo",
		IncludePrerelease: true,
		BaseURL:           server.URL,
		HTTPClient:        server.Client(),
	}).Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if withPrerelease.Version != "v2.0.0-rc.1" || !withPrerelease.Prerelease {
		t.Fatalf("prerelease result = %+v", withPrerelease)
	}
}

func TestGitHubProviderReturnsNoRelease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[{"tag_name":"not-semver","draft":false,"prerelease":false,"assets":[]}]`)
	}))
	defer server.Close()

	_, err := (GitHubProvider{
		Owner:      "acme",
		Repository: "demo",
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}).Latest(context.Background())
	if err != ErrNoRelease {
		t.Fatalf("expected ErrNoRelease, got %v", err)
	}
}
