package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigParsesProfilesAndRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := `
server:
  listen: ":9090"

svgs:
  default:
    width: 640
    font_size: 15
    rows:
      - type: text
        text: "Hello: world"
      - type: header
        name: "X-Request-ID"
        label: "Request ID"
      - type: query
        label: "All Query"
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.Server.Listen != ":9090" {
		t.Fatalf("listen = %q, want :9090", cfg.Server.Listen)
	}
	svg := cfg.SVGs["default"]
	if svg.Width != 640 || svg.FontSize != 15 {
		t.Fatalf("svg dimensions = %dx%d font, want width 640 font 15", svg.Width, svg.FontSize)
	}
	if len(svg.Rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(svg.Rows))
	}
	if svg.Rows[0].Text != "Hello: world" {
		t.Fatalf("text row = %q", svg.Rows[0].Text)
	}
	if svg.Rows[2].Type != "query" || svg.Rows[2].Name != "" || svg.Rows[2].Label != "All Query" {
		t.Fatalf("query-all row parsed incorrectly: %+v", svg.Rows[2])
	}
}

func TestLoadConfigAppliesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := `
svgs:
  default:
    rows:
      - type: text
        text: "Hello"
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.Server.Listen != defaultListen {
		t.Fatalf("listen = %q, want %q", cfg.Server.Listen, defaultListen)
	}
	if cfg.SVGs["default"].Width != defaultSVGWidth {
		t.Fatalf("width = %d, want %d", cfg.SVGs["default"].Width, defaultSVGWidth)
	}
	if cfg.SVGs["default"].FontSize != defaultFontSize {
		t.Fatalf("font_size = %d, want %d", cfg.SVGs["default"].FontSize, defaultFontSize)
	}
}

func TestLoadConfigSupportsListenEnvOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := `
server:
  listen: ":8080"
svgs:
  default:
    rows:
      - type: text
        text: "Hello"
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv(listenEnv, ":8081")

	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}

	if cfg.Server.Listen != ":8081" {
		t.Fatalf("listen = %q, want :8081", cfg.Server.Listen)
	}
}

func TestLoadConfigRejectsInvalidProfileName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	data := `
svgs:
  bad/name:
    rows:
      - type: text
        text: "Hello"
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, err := loadConfig(path); err == nil {
		t.Fatal("loadConfig() error = nil, want invalid profile error")
	}
}
