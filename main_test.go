package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writeVersions(t *testing.T, body string) {
	t.Helper()
	p := filepath.Join(t.TempDir(), "versions.json")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	versionsPath = p
}

func TestVersionListShapeAndDefaults(t *testing.T) {
	writeVersions(t, `{"titles":[{"id":1,"version":100},{"id":2,"version":200,"required_version":150}]}`)
	body, etag, err := buildVersionList(versionsPath)
	if err != nil {
		t.Fatal(err)
	}
	var vl versionList
	if err := json.Unmarshal(body, &vl); err != nil {
		t.Fatal(err)
	}
	if vl.FormatVersion != 1 || vl.LastModified == 0 || len(vl.Titles) != 2 {
		t.Fatalf("shape %+v", vl)
	}
	// required_version defaults to version when omitted
	if vl.Titles[0].RequiredVersion != 100 {
		t.Fatalf("default required_version %d", vl.Titles[0].RequiredVersion)
	}
	if vl.Titles[1].RequiredVersion != 150 {
		t.Fatalf("explicit required_version %d", vl.Titles[1].RequiredVersion)
	}
	if etag == "" {
		t.Fatal("no etag")
	}
}

func TestIfNoneMatch304(t *testing.T) {
	writeVersions(t, `{"titles":[{"id":1,"version":100}]}`)
	_, etag, _ := buildVersionList(versionsPath)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/tagaya/hac_versionlist", nil)
	req.Header.Set("If-None-Match", etag)
	handleVersionList(rec, req)
	if rec.Code != 304 {
		t.Fatalf("matching ETag should 304, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	handleVersionList(rec, httptest.NewRequest("GET", "/tagaya/hac_versionlist", nil))
	if rec.Code != 200 || rec.Header().Get("ETag") == "" {
		t.Fatalf("plain GET code %d etag %q", rec.Code, rec.Header().Get("ETag"))
	}
}

// TestEmbeddedFallback: with no versions.json on disk, buildVersionList must use
// the embedded default and return a valid list, not an error (regression for the
// server 500-ing every request on a fresh deploy).
func TestEmbeddedFallback(t *testing.T) {
	versionsPath = filepath.Join(t.TempDir(), "does-not-exist.json")
	body, etag, err := buildVersionList(versionsPath)
	if err != nil {
		t.Fatalf("fallback errored: %v", err)
	}
	var vl versionList
	if err := json.Unmarshal(body, &vl); err != nil {
		t.Fatal(err)
	}
	if vl.FormatVersion != 1 || len(vl.Titles) == 0 || etag == "" {
		t.Fatalf("embedded fallback produced empty list: %+v", vl)
	}
}
