package main

import (
	"bytes"
	"html"
	"regexp"
	"strings"
	"text/template"
)

type PostData struct {
	Title       string
	Description string // HTML-stripped, entity-decoded
	Link        string
	PubDate     string
}

var (
	htmlTagRe    = regexp.MustCompile(`<[^>]+>`)
	whitespaceRe = regexp.MustCompile(`[ \t]+`)
	newlineRe    = regexp.MustCompile(`\n{3,}`)
)

func buildPostText(tmplStr string, item *RSSItem, charLimit int) (string, error) {
	tmpl, err := template.New("post").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	desc := processDescription(item.Description)

	if charLimit > 0 {
		// Render the template with an empty description to measure the fixed
		// overhead (link, separators, static text). This ensures the description
		// is truncated to fit rather than the link being silently dropped.
		shell, err := renderTemplate(tmpl, PostData{
			Title:   item.Title,
			Link:    item.Link,
			PubDate: item.PubDate,
		})
		if err != nil {
			return "", err
		}
		overhead := len([]rune(normalizeWhitespace(shell)))
		budget := charLimit - overhead
		if budget > 0 {
			desc = truncate(desc, budget)
			// Remove trailing ellipsis whitespace that truncate may leave so the
			// shell join below doesn't produce a double-space before the link.
		}
	}

	text, err := renderTemplate(tmpl, PostData{
		Title:       item.Title,
		Description: desc,
		Link:        item.Link,
		PubDate:     item.PubDate,
	})
	if err != nil {
		return "", err
	}

	return normalizeWhitespace(text), nil
}

func renderTemplate(tmpl *template.Template, data PostData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func processDescription(raw string) string {
	// Strip HTML tags
	stripped := htmlTagRe.ReplaceAllString(raw, "")
	// Decode HTML entities
	decoded := html.UnescapeString(stripped)
	return decoded
}

func normalizeWhitespace(s string) string {
	// Normalize line endings
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	// Collapse horizontal whitespace within lines
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(whitespaceRe.ReplaceAllString(line, " "))
	}
	s = strings.Join(lines, "\n")
	// Collapse runs of 3+ newlines to 2
	s = newlineRe.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

func truncate(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}

	// Reserve one rune for the ellipsis
	cut := limit - 1
	// Walk back to a word boundary
	for cut > 0 && !isWordBoundary(runes[cut]) {
		cut--
	}
	if cut == 0 {
		// No word boundary found; hard truncate
		cut = limit - 1
	}

	return strings.TrimRight(string(runes[:cut]), " \t\n") + "…"
}

func isWordBoundary(r rune) bool {
	return r == ' ' || r == '\n' || r == '\t'
}
