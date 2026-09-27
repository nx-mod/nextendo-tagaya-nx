// Command nextendo-tagaya-nx serves Tagaya, the Switch title-version-list service,
// for Nextendo Network.
//
// A console asks Tagaya for the latest version of each title, so it knows whether
// an update exists and what minimum version online play requires. The real host
// is tagaya.hac.lp1.eshop.nintendo.net; sni-router sends it here. Without an
// answer a game can nag "software update required" or gate online on
// required_version, so serving a version list the stack controls keeps titles
// playable.
//
//	GET /tagaya/hac_versionlist   ->  { format_version, last_modified, titles[] }
//
// The list comes from versions.json (hot-reloaded); an ETag + If-None-Match give
// the console a cheap 304 when nothing changed.
package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

var (
	httpPort     = envOrInt("TAGAYA_PORT", 8471)
	versionsPath = envOr("TAGAYA_VERSIONS", "versions.json")
	certFile     = envOr("CERT_FILE", "")
	keyFile      = envOr("KEY_FILE", "")
	dashPort     = envOr("DASH_PORT", "8100")
	dashToken    = envOr("DASH_TOKEN", "")

	reqTotal  atomic.Int64
	notMod    atomic.Int64
	dashStart = time.Now()
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func envOrInt(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return d
}

func main() {
	log.SetOutput(os.Stdout)

	mux := http.NewServeMux()
	mux.HandleFunc("/tagaya/hac_versionlist", handleVersionList)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "ok") })
	go startDashboard()

	addr := fmt.Sprintf(":%d", httpPort)
	srv := &http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 15 * time.Second}
	if certFile != "" && keyFile != "" {
		log.Printf("[Tagaya] listening HTTPS %s (versions=%s)", addr, versionsPath)
		log.Fatal(srv.ListenAndServeTLS(certFile, keyFile))
	}
	log.Printf("[Tagaya] listening HTTP %s (TLS via sni-router; versions=%s)", addr, versionsPath)
	log.Fatal(srv.ListenAndServe())
}

// handleVersionList answers GET /tagaya/hac_versionlist.
func handleVersionList(w http.ResponseWriter, r *http.Request) {
	reqTotal.Add(1)
	body, etag, err := buildVersionList(versionsPath)
	if err != nil {
		log.Printf("[Tagaya] %s: %v", versionsPath, err)
		http.Error(w, "version list unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("ETag", etag)
	if match := r.Header.Get("If-None-Match"); match == etag {
		notMod.Add(1)
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write(body)
}

// buildVersionList renders versions.json into the Tagaya response and its ETag.
func buildVersionList(path string) ([]byte, string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var cfg struct {
		LastModified int64   `json:"last_modified"`
		Titles       []title `json:"titles"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, "", err
	}
	if cfg.LastModified == 0 {
		cfg.LastModified = time.Now().Unix()
	}
	out := versionList{FormatVersion: 1, LastModified: cfg.LastModified, Titles: normalize(cfg.Titles)}
	body, err := json.Marshal(out)
	if err != nil {
		return nil, "", err
	}
	sum := sha1.Sum(body)
	return body, fmt.Sprintf("\"%x\"", sum[:8]), nil
}

type title struct {
	ID              uint64 `json:"id"`
	Version         uint32 `json:"version"`
	RequiredVersion uint32 `json:"required_version"`
}

type versionList struct {
	FormatVersion int     `json:"format_version"`
	LastModified  int64   `json:"last_modified"`
	Titles        []title `json:"titles"`
}

// normalize fills required_version from version when it is omitted (a title with
// no minimum for online play just requires whatever it ships).
func normalize(ts []title) []title {
	for i := range ts {
		if ts[i].RequiredVersion == 0 {
			ts[i].RequiredVersion = ts[i].Version
		}
	}
	return ts
}

func startDashboard() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		if dashToken != "" && r.URL.Query().Get("key") != dashToken {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		n := 0
		if _, ts, err := loadTitles(versionsPath); err == nil {
			n = ts
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"uptimeSeconds": int(time.Since(dashStart).Seconds()),
			"requests":      reqTotal.Load(),
			"notModified":   notMod.Load(),
			"titles":        n,
			"stack":         "tagaya",
		})
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintln(w, "ok") })
	log.Printf("[Tagaya Dashboard] :%s", dashPort)
	if err := http.ListenAndServe(":"+dashPort, mux); err != nil {
		log.Printf("[Tagaya Dashboard] %v", err)
	}
}

func loadTitles(path string) ([]byte, int, error) {
	body, _, err := buildVersionList(path)
	if err != nil {
		return nil, 0, err
	}
	var vl versionList
	_ = json.Unmarshal(body, &vl)
	return body, len(vl.Titles), nil
}
