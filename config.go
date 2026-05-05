package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

const (
	defaultConfigPath = "config.yaml"
	defaultListen     = ":8080"
	defaultSVGWidth   = 800
	defaultFontSize   = 16

	configPathEnv = "SVG_REFLECT_CONFIG"
	listenEnv     = "SVG_REFLECT_LISTEN"
)

var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type Config struct {
	Server ServerConfig
	SVGs   map[string]SVGConfig
}

type ServerConfig struct {
	Listen string
}

type SVGConfig struct {
	Width    int
	FontSize int
	Rows     []RowConfig
}

type RowConfig struct {
	Type  string
	Name  string
	Label string
	Text  string
}

func configPathFromEnv() string {
	if path := os.Getenv(configPathEnv); path != "" {
		return path
	}
	return defaultConfigPath
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg, err := parseConfigYAML(string(data))
	if err != nil {
		return nil, err
	}
	applyConfigDefaults(cfg)
	applyConfigEnvOverrides(cfg)
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func applyConfigDefaults(cfg *Config) {
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = defaultListen
	}
	for name, svg := range cfg.SVGs {
		if svg.Width == 0 {
			svg.Width = defaultSVGWidth
		}
		if svg.FontSize == 0 {
			svg.FontSize = defaultFontSize
		}
		cfg.SVGs[name] = svg
	}
}

func applyConfigEnvOverrides(cfg *Config) {
	if listen := os.Getenv(listenEnv); listen != "" {
		cfg.Server.Listen = listen
	}
}

func validateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}
	if len(cfg.SVGs) == 0 {
		return fmt.Errorf("svgs must define at least one profile")
	}
	for name, svg := range cfg.SVGs {
		if !profileNamePattern.MatchString(name) {
			return fmt.Errorf("svg profile %q must match %s", name, profileNamePattern.String())
		}
		if svg.Width <= 0 {
			return fmt.Errorf("svg profile %q width must be positive", name)
		}
		if svg.FontSize <= 0 {
			return fmt.Errorf("svg profile %q font_size must be positive", name)
		}
		for i, row := range svg.Rows {
			switch row.Type {
			case "text":
			case "header":
				if row.Name == "" {
					return fmt.Errorf("svg profile %q row %d header name is required", name, i)
				}
			case "query":
			default:
				return fmt.Errorf("svg profile %q row %d has unsupported type %q", name, i, row.Type)
			}
		}
	}
	return nil
}

type yamlLine struct {
	indent int
	text   string
	line   int
}

func parseConfigYAML(data string) (*Config, error) {
	lines, err := preprocessYAMLLines(data)
	if err != nil {
		return nil, err
	}

	cfg := &Config{SVGs: make(map[string]SVGConfig)}
	for i := 0; i < len(lines); {
		line := lines[i]
		if line.indent != 0 {
			return nil, fmt.Errorf("line %d: top-level keys must not be indented", line.line)
		}

		key, value, ok := splitYAMLMapping(line.text)
		if !ok || value != "" {
			return nil, fmt.Errorf("line %d: expected top-level section", line.line)
		}

		switch key {
		case "server":
			var err error
			i, err = parseServerSection(lines, i+1, cfg)
			if err != nil {
				return nil, err
			}
		case "svgs":
			var err error
			i, err = parseSVGsSection(lines, i+1, cfg)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("line %d: unknown top-level section %q", line.line, key)
		}
	}
	return cfg, nil
}

func preprocessYAMLLines(data string) ([]yamlLine, error) {
	var lines []yamlLine
	for lineNo, raw := range strings.Split(data, "\n") {
		raw = strings.TrimRight(strings.TrimSuffix(raw, "\r"), " \t")
		raw = stripInlineComment(raw)
		if strings.TrimSpace(raw) == "" {
			continue
		}

		indent := 0
		for indent < len(raw) {
			switch raw[indent] {
			case ' ':
				indent++
			case '\t':
				return nil, fmt.Errorf("line %d: tabs are not supported for indentation", lineNo+1)
			default:
				goto doneIndent
			}
		}
	doneIndent:
		lines = append(lines, yamlLine{
			indent: indent,
			text:   strings.TrimSpace(raw),
			line:   lineNo + 1,
		})
	}
	return lines, nil
}

func stripInlineComment(line string) string {
	inSingle := false
	inDouble := false
	escaped := false

	for i, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if inDouble && r == '\\' {
			escaped = true
			continue
		}
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case '#':
			if !inSingle && !inDouble && (i == 0 || line[i-1] == ' ' || line[i-1] == '\t') {
				return strings.TrimRight(line[:i], " \t")
			}
		}
	}
	return line
}

func parseServerSection(lines []yamlLine, i int, cfg *Config) (int, error) {
	for i < len(lines) && lines[i].indent > 0 {
		line := lines[i]
		if line.indent != 2 {
			return 0, fmt.Errorf("line %d: server fields must use two-space indentation", line.line)
		}

		key, value, ok := splitYAMLMapping(line.text)
		if !ok {
			return 0, fmt.Errorf("line %d: expected server field", line.line)
		}
		switch key {
		case "listen":
			var err error
			cfg.Server.Listen, err = parseYAMLScalar(value)
			if err != nil {
				return 0, fmt.Errorf("line %d: %w", line.line, err)
			}
		default:
			return 0, fmt.Errorf("line %d: unknown server field %q", line.line, key)
		}
		i++
	}
	return i, nil
}

func parseSVGsSection(lines []yamlLine, i int, cfg *Config) (int, error) {
	for i < len(lines) && lines[i].indent > 0 {
		line := lines[i]
		if line.indent != 2 {
			return 0, fmt.Errorf("line %d: svg profile names must use two-space indentation", line.line)
		}

		profileName, value, ok := splitYAMLMapping(line.text)
		if !ok || value != "" {
			return 0, fmt.Errorf("line %d: expected svg profile section", line.line)
		}

		svg := SVGConfig{}
		i++
		for i < len(lines) && lines[i].indent > 2 {
			fieldLine := lines[i]
			if fieldLine.indent != 4 {
				return 0, fmt.Errorf("line %d: svg profile fields must use four-space indentation", fieldLine.line)
			}

			key, value, ok := splitYAMLMapping(fieldLine.text)
			if !ok {
				return 0, fmt.Errorf("line %d: expected svg profile field", fieldLine.line)
			}

			switch key {
			case "width":
				width, err := parseYAMLInt(value)
				if err != nil {
					return 0, fmt.Errorf("line %d: %w", fieldLine.line, err)
				}
				svg.Width = width
				i++
			case "font_size":
				fontSize, err := parseYAMLInt(value)
				if err != nil {
					return 0, fmt.Errorf("line %d: %w", fieldLine.line, err)
				}
				svg.FontSize = fontSize
				i++
			case "rows":
				if value != "" {
					return 0, fmt.Errorf("line %d: rows must be a list block", fieldLine.line)
				}
				rows, next, err := parseRows(lines, i+1)
				if err != nil {
					return 0, err
				}
				svg.Rows = rows
				i = next
			default:
				return 0, fmt.Errorf("line %d: unknown svg profile field %q", fieldLine.line, key)
			}
		}
		cfg.SVGs[profileName] = svg
	}
	return i, nil
}

func parseRows(lines []yamlLine, i int) ([]RowConfig, int, error) {
	var rows []RowConfig
	for i < len(lines) && lines[i].indent > 4 {
		line := lines[i]
		if line.indent != 6 {
			return nil, 0, fmt.Errorf("line %d: rows must use six-space indentation", line.line)
		}
		if line.text != "-" && !strings.HasPrefix(line.text, "- ") {
			return nil, 0, fmt.Errorf("line %d: expected row list item", line.line)
		}

		row := RowConfig{}
		itemText := ""
		if strings.HasPrefix(line.text, "- ") {
			itemText = strings.TrimSpace(strings.TrimPrefix(line.text, "- "))
		}
		if itemText != "" {
			key, value, ok := splitYAMLMapping(itemText)
			if !ok {
				return nil, 0, fmt.Errorf("line %d: expected row field after '-'", line.line)
			}
			if err := setRowField(&row, key, value, line.line); err != nil {
				return nil, 0, err
			}
		}

		i++
		for i < len(lines) && lines[i].indent > 6 {
			fieldLine := lines[i]
			if fieldLine.indent != 8 {
				return nil, 0, fmt.Errorf("line %d: row fields must use eight-space indentation", fieldLine.line)
			}

			key, value, ok := splitYAMLMapping(fieldLine.text)
			if !ok {
				return nil, 0, fmt.Errorf("line %d: expected row field", fieldLine.line)
			}
			if err := setRowField(&row, key, value, fieldLine.line); err != nil {
				return nil, 0, err
			}
			i++
		}
		rows = append(rows, row)
	}
	return rows, i, nil
}

func setRowField(row *RowConfig, key, value string, line int) error {
	scalar, err := parseYAMLScalar(value)
	if err != nil {
		return fmt.Errorf("line %d: %w", line, err)
	}
	switch key {
	case "type":
		row.Type = scalar
	case "name":
		row.Name = scalar
	case "label":
		row.Label = scalar
	case "text":
		row.Text = scalar
	default:
		return fmt.Errorf("line %d: unknown row field %q", line, key)
	}
	return nil
}

func splitYAMLMapping(text string) (string, string, bool) {
	inSingle := false
	inDouble := false
	escaped := false

	for i, r := range text {
		if escaped {
			escaped = false
			continue
		}
		if inDouble && r == '\\' {
			escaped = true
			continue
		}
		switch r {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case ':':
			if !inSingle && !inDouble {
				key := strings.TrimSpace(text[:i])
				value := strings.TrimSpace(text[i+1:])
				return key, value, key != ""
			}
		}
	}
	return "", "", false
}

func parseYAMLScalar(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, `"`) || strings.HasSuffix(value, `"`) {
		if !(strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`)) {
			return "", fmt.Errorf("invalid double-quoted scalar %q", value)
		}
		unquoted, err := strconv.Unquote(value)
		if err != nil {
			return "", fmt.Errorf("invalid double-quoted scalar %q", value)
		}
		return unquoted, nil
	}
	if strings.HasPrefix(value, `'`) || strings.HasSuffix(value, `'`) {
		if !(strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`)) {
			return "", fmt.Errorf("invalid single-quoted scalar %q", value)
		}
		return strings.ReplaceAll(value[1:len(value)-1], "''", "'"), nil
	}
	return value, nil
}

func parseYAMLInt(value string) (int, error) {
	scalar, err := parseYAMLScalar(value)
	if err != nil {
		return 0, err
	}
	parsed, err := strconv.Atoi(scalar)
	if err != nil {
		return 0, fmt.Errorf("expected integer, got %q", scalar)
	}
	return parsed, nil
}
