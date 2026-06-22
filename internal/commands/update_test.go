package commands

import (
	"runtime"
	"strings"
	"testing"
)

func TestFindReleaseAssetUsesPublishedAsset(t *testing.T) {
	t.Parallel()

	name := "qq_" + runtime.GOOS + "_" + runtime.GOARCH + ".tar.gz"
	release := githubRelease{TagName: "v1.2.3"}
	release.Assets = append(release.Assets, struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	}{Name: name, BrowserDownloadURL: "https://example.test/" + name})

	url, gotName, err := findReleaseAsset(release)
	if err != nil {
		t.Fatal(err)
	}
	if gotName != name || !strings.HasSuffix(url, name) {
		t.Fatalf("unexpected asset: %q %q", url, gotName)
	}
}

func TestPathBase(t *testing.T) {
	t.Parallel()
	if got := pathBase("https://github.com/example/releases/tag/v1.2.3/"); got != "v1.2.3" {
		t.Fatalf("pathBase() = %q", got)
	}
}
