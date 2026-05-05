package main

import (
	"net/http"
	"strings"
)

func newHandler(cfg *Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/svg/", func(w http.ResponseWriter, r *http.Request) {
		handleSVG(w, r, cfg)
	})
	return mux
}

func handleSVG(w http.ResponseWriter, r *http.Request, cfg *Config) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	profile, ok := svgProfileFromPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}

	svg, ok := cfg.SVGs[profile]
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(renderSVG(svg, r)))
}

func svgProfileFromPath(path string) (string, bool) {
	const prefix = "/svg/"
	const suffix = ".svg"

	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return "", false
	}
	profile := strings.TrimSuffix(strings.TrimPrefix(path, prefix), suffix)
	if profile == "" || strings.Contains(profile, "/") {
		return "", false
	}
	if !profileNamePattern.MatchString(profile) {
		return "", false
	}
	return profile, true
}
