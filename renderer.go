package main

import (
	"fmt"
	"html"
	"net/http"
	"sort"
	"strings"
)

const (
	svgPaddingX             = 24
	svgPaddingY             = 18
	monospaceCharWidthNumer = 6
	monospaceCharWidthDenom = 10
	continuationPrefix      = "  "
)

func renderSVG(svg SVGConfig, req *http.Request) string {
	lines := wrapRenderedLines(renderLines(svg, req), svg.Width, svg.FontSize)
	lineHeight := svg.FontSize + svg.FontSize/2
	if lineHeight < svg.FontSize+6 {
		lineHeight = svg.FontSize + 6
	}

	height := svgPaddingY*2 + lineHeight*len(lines)
	if height < 64 {
		height = 64
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<?xml version="1.0" encoding="UTF-8"?>`+"\n")
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`+"\n", svg.Width, height, svg.Width, height)
	fmt.Fprintf(&b, `  <rect width="100%%" height="100%%" fill="#ffffff"/>`+"\n")
	fmt.Fprintf(&b, `  <g font-family="ui-monospace, SFMono-Regular, Consolas, Liberation Mono, monospace" font-size="%d" fill="#111827">`+"\n", svg.FontSize)
	for i, line := range lines {
		y := svgPaddingY + svg.FontSize + i*lineHeight
		fmt.Fprintf(&b, `    <text x="%d" y="%d">%s</text>`+"\n", svgPaddingX, y, html.EscapeString(line))
	}
	fmt.Fprintf(&b, "  </g>\n")
	fmt.Fprintf(&b, "</svg>\n")
	return b.String()
}

func renderLines(svg SVGConfig, req *http.Request) []string {
	var lines []string
	query := req.URL.Query()

	for _, row := range svg.Rows {
		switch row.Type {
		case "text":
			lines = append(lines, row.Text)
		case "header":
			label := labelOrName(row.Label, row.Name)
			lines = append(lines, formatKV(label, strings.Join(requestHeaderValues(req.Header, row.Name), ", ")))
		case "query":
			if row.Name != "" {
				label := labelOrName(row.Label, row.Name)
				lines = append(lines, formatKV(label, strings.Join(query[row.Name], ", ")))
				continue
			}
			lines = append(lines, renderAllQueryLines(row.Label, query)...)
		}
	}

	return lines
}

func renderAllQueryLines(label string, query map[string][]string) []string {
	if len(query) == 0 {
		if label == "" {
			label = "query"
		}
		return []string{formatKV(label, "")}
	}

	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		displayName := key
		if label != "" {
			displayName = label + "." + key
		}
		lines = append(lines, formatKV(displayName, strings.Join(query[key], ", ")))
	}
	return lines
}

func requestHeaderValues(headers http.Header, name string) []string {
	var values []string
	for key, headerValues := range headers {
		if strings.EqualFold(key, name) {
			values = append(values, headerValues...)
		}
	}
	return values
}

func labelOrName(label, name string) string {
	if label != "" {
		return label
	}
	return name
}

func formatKV(label, value string) string {
	return label + ": " + value
}

func wrapRenderedLines(lines []string, width, fontSize int) []string {
	maxChars := maxLineChars(width, fontSize)
	wrapped := make([]string, 0, len(lines))
	for _, line := range lines {
		wrapped = append(wrapped, wrapLine(line, maxChars)...)
	}
	return wrapped
}

func maxLineChars(width, fontSize int) int {
	if fontSize <= 0 {
		fontSize = defaultFontSize
	}
	contentWidth := width - svgPaddingX*2
	if contentWidth <= 0 {
		return 1
	}
	maxChars := contentWidth * monospaceCharWidthDenom / (fontSize * monospaceCharWidthNumer)
	if maxChars < 1 {
		return 1
	}
	return maxChars
}

func wrapLine(line string, maxChars int) []string {
	if maxChars < 1 {
		maxChars = 1
	}

	remaining := []rune(line)
	if len(remaining) <= maxChars {
		return []string{line}
	}

	var lines []string
	prefix := ""
	for len(remaining) > 0 {
		available := maxChars - len([]rune(prefix))
		if available < 1 {
			available = maxChars
			prefix = ""
		}
		if len(remaining) <= available {
			lines = append(lines, prefix+string(remaining))
			break
		}

		splitAt := bestWrapSplit(remaining, available)
		chunk := strings.TrimRight(string(remaining[:splitAt]), " ")
		lines = append(lines, prefix+chunk)

		remaining = trimLeftSpaces(remaining[splitAt:])
		prefix = continuationPrefix
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func bestWrapSplit(line []rune, limit int) int {
	if limit >= len(line) {
		return len(line)
	}
	for i := limit - 1; i > 0; i-- {
		if isPreferredWrapRune(line[i]) {
			return i + 1
		}
	}
	return limit
}

func isPreferredWrapRune(r rune) bool {
	switch r {
	case ' ', ',', ';', '&', '?', '/', '\\':
		return true
	default:
		return false
	}
}

func trimLeftSpaces(line []rune) []rune {
	for len(line) > 0 && line[0] == ' ' {
		line = line[1:]
	}
	return line
}
