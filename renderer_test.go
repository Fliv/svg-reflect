package main

import (
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestRenderSVGIncludesConfiguredRows(t *testing.T) {
	svg := SVGConfig{
		Width:    800,
		FontSize: 16,
		Rows: []RowConfig{
			{Type: "text", Text: "Custom static content"},
			{Type: "header", Name: "X-Request-ID", Label: "Request ID"},
			{Type: "query", Name: "user", Label: "User"},
			{Type: "query", Label: "All Query"},
		},
	}
	req := httptest.NewRequest("GET", "/svg/default.svg?b=2&a=1&a=3&user=alice", nil)
	req.Header.Add("x-request-id", "rid-1")

	out := renderSVG(svg, req)

	for _, want := range []string{
		"Custom static content",
		"Request ID: rid-1",
		"User: alice",
		"All Query.a: 1, 3",
		"All Query.b: 2",
		"All Query.user: alice",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered SVG missing %q:\n%s", want, out)
		}
	}

	aIndex := strings.Index(out, "All Query.a: 1, 3")
	bIndex := strings.Index(out, "All Query.b: 2")
	if aIndex < 0 || bIndex < 0 || aIndex > bIndex {
		t.Fatalf("all query keys were not rendered in sorted order:\n%s", out)
	}
}

func TestRenderSVGEscapesTextAndValues(t *testing.T) {
	svg := SVGConfig{
		Width:    800,
		FontSize: 16,
		Rows: []RowConfig{
			{Type: "text", Text: `<custom>&"`},
			{Type: "header", Name: "X-Danger"},
			{Type: "query", Name: "q"},
		},
	}
	req := httptest.NewRequest("GET", `/svg/default.svg?q=<query>&"`, nil)
	req.Header.Set("X-Danger", `<header>&"`)

	out := renderSVG(svg, req)

	for _, want := range []string{
		"&lt;custom&gt;&amp;&#34;",
		"X-Danger: &lt;header&gt;&amp;&#34;",
		"q: &lt;query&gt;",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered SVG missing escaped text %q:\n%s", want, out)
		}
	}
}

func TestRenderSVGShowsEmptyMissingValues(t *testing.T) {
	svg := SVGConfig{
		Width:    800,
		FontSize: 16,
		Rows: []RowConfig{
			{Type: "header", Name: "X-Missing", Label: "Missing Header"},
			{Type: "query", Name: "missing", Label: "Missing Query"},
			{Type: "query", Label: "All Query"},
		},
	}
	req := httptest.NewRequest("GET", "/svg/default.svg", nil)

	out := renderSVG(svg, req)

	for _, want := range []string{
		"Missing Header: ",
		"Missing Query: ",
		"All Query: ",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("rendered SVG missing %q:\n%s", want, out)
		}
	}
}

func TestRenderSVGWrapsLongLinesAndGrowsHeight(t *testing.T) {
	svg := SVGConfig{
		Width:    170,
		FontSize: 16,
		Rows: []RowConfig{
			{Type: "header", Name: "X-Long", Label: "Long"},
		},
	}
	req := httptest.NewRequest("GET", "/svg/default.svg", nil)
	req.Header.Set("X-Long", "abcdefghijklmnopqrstuvwxyz0123456789")

	out := renderSVG(svg, req)

	if count := strings.Count(out, "<text "); count < 4 {
		t.Fatalf("rendered SVG text elements = %d, want wrapped long line:\n%s", count, out)
	}
	if strings.Contains(out, "abcdefghijklmnopqrstuvwxyz0123456789") {
		t.Fatalf("long value was not wrapped:\n%s", out)
	}

	heightMatch := regexp.MustCompile(`height="(\d+)"`).FindStringSubmatch(out)
	if len(heightMatch) != 2 || heightMatch[1] == "64" {
		t.Fatalf("SVG height did not grow after wrapping:\n%s", out)
	}
}

func TestWrapLineHonorsMaxChars(t *testing.T) {
	lines := wrapLine("Token: abcdefghijklmnopqrstuvwxyz", 12)
	if len(lines) < 3 {
		t.Fatalf("wrapped lines = %d, want at least 3", len(lines))
	}
	for _, line := range lines {
		if got := len([]rune(line)); got > 12 {
			t.Fatalf("line %q length = %d, want <= 12", line, got)
		}
	}
}
