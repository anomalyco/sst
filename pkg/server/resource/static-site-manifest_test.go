package resource

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStaticSiteManifestResolve(t *testing.T) {
	outputPath := t.TempDir()
	err := os.MkdirAll(filepath.Join(outputPath, "assets"), 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.MkdirAll(filepath.Join(outputPath, ".well-known"), 0o755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(outputPath, "index.html"), []byte("<html></html>"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(outputPath, "assets", "app.js"), []byte("console.log(1)"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(outputPath, ".well-known", "apple-app-site-association"), []byte("{}"), 0o644)
	if err != nil {
		t.Fatal(err)
	}

	manifest := &StaticSiteManifest{run: NewRun()}
	outs, err := manifest.resolve(&StaticSiteManifestInputs{
		SitePath:     outputPath,
		OutputPath:   outputPath,
		Environment:  map[string]string{},
		FileOptions:  []StaticSiteManifestFileOption{{Files: []string{"**"}}},
		TextEncoding: "utf-8",
		KeyPrefix:    ptrStaticSiteString("docs"),
		AssetPath:    ptrStaticSiteString("site-assets"),
		AssetRoutes:  []string{"/uploads"},
		BucketDomain: ptrStaticSiteString("bucket.example.com"),
		ErrorPage:    ptrStaticSiteString("/404.html"),
		Base:         ptrStaticSiteString("/docs"),
		Trigger:      "1",
	})
	if err != nil {
		t.Fatal(err)
	}

	keys := []string{}
	for _, file := range outs.Files {
		keys = append(keys, file.Key)
		if len(file.Hash) != 64 {
			t.Fatalf("expected sha256 hash, got %q", file.Hash)
		}
	}
	expected := []string{
		"docs/.well-known/apple-app-site-association",
		"docs/assets/app.js",
		"docs/index.html",
	}
	if len(keys) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, keys)
	}
	for i, item := range expected {
		if keys[i] != item {
			t.Fatalf("expected %v, got %v", expected, keys)
		}
	}

	if _, ok := outs.AssetManifest["index.html"]; !ok {
		t.Fatal("expected index.html in asset manifest")
	}
	if outs.KvEntries["/assets/app.js"] != "s3" {
		t.Fatalf("expected kv entry for /assets/app.js, got %q", outs.KvEntries["/assets/app.js"])
	}

	metadata := map[string]any{}
	err = json.Unmarshal([]byte(outs.KvEntries["metadata"]), &metadata)
	if err != nil {
		t.Fatal(err)
	}
	if metadata["base"] != "/docs" {
		t.Fatalf("expected base /docs, got %v", metadata["base"])
	}
	s3 := metadata["s3"].(map[string]any)
	if s3["dir"] != "/site-assets" {
		t.Fatalf("expected dir /site-assets, got %v", s3["dir"])
	}
	routes := s3["routes"].([]any)
	if len(routes) != 2 || routes[0] != "/uploads" || routes[1] != "/assets" {
		t.Fatalf("unexpected routes %v", routes)
	}
}

func TestStaticSiteManifestMissingOutput(t *testing.T) {
	manifest := &StaticSiteManifest{run: NewRun()}
	_, err := manifest.resolve(&StaticSiteManifestInputs{
		SitePath:     t.TempDir(),
		OutputPath:   filepath.Join(t.TempDir(), "dist"),
		Environment:  map[string]string{},
		FileOptions:  []StaticSiteManifestFileOption{},
		TextEncoding: "utf-8",
		Trigger:      "1",
	})
	if err == nil || err.Error() == "" {
		t.Fatal("expected missing output error")
	}
	if want := "No build output found"; !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error to start with %q, got %q", want, err.Error())
	}
}

func ptrStaticSiteString(value string) *string {
	return &value
}
