package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesDifferentProfiles(t *testing.T) {
	cfg := &Config{SVGs: map[string]SVGConfig{
		"default": {
			Width:    800,
			FontSize: 16,
			Rows:     []RowConfig{{Type: "text", Text: "Default SVG"}},
		},
		"debug": {
			Width:    800,
			FontSize: 16,
			Rows:     []RowConfig{{Type: "text", Text: "Debug SVG"}},
		},
	}}
	handler := newHandler(cfg)

	for path, want := range map[string]string{
		"/svg/default.svg": "Default SVG",
		"/svg/debug.svg":   "Debug SVG",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "image/svg+xml; charset=utf-8" {
			t.Fatalf("%s Content-Type = %q", path, ct)
		}
		if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
			t.Fatalf("%s Cache-Control = %q", path, cc)
		}
		if !strings.Contains(rec.Body.String(), want) {
			t.Fatalf("%s body missing %q:\n%s", path, want, rec.Body.String())
		}
	}
}

func TestHandlerReturns404ForUnknownOrInvalidProfiles(t *testing.T) {
	cfg := &Config{SVGs: map[string]SVGConfig{
		"default": {Width: 800, FontSize: 16},
	}}
	handler := newHandler(cfg)

	for _, path := range []string{
		"/svg/missing.svg",
		"/svg/bad/name.svg",
		"/svg/bad!.svg",
		"/svg/default.png",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want 404", path, rec.Code)
		}
	}
}

func TestHandlerRejectsNonGET(t *testing.T) {
	cfg := &Config{SVGs: map[string]SVGConfig{
		"default": {Width: 800, FontSize: 16},
	}}
	handler := newHandler(cfg)

	req := httptest.NewRequest(http.MethodPost, "/svg/default.svg", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("Allow = %q, want GET", allow)
	}
}
