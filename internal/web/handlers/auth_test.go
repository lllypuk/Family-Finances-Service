package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeRedirectURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", "/"},
		{"root", "/", "/"},
		{"valid path", "/dashboard", "/dashboard"},
		{"valid path with query", "/page?id=1", "/page?id=1"},
		{"valid nested path", "/admin/users", "/admin/users"},
		{"valid with multiple query params", "/search?q=test&page=2", "/search?q=test&page=2"},
		{"protocol relative", "//evil.com", "/"},
		{"absolute http", "http://evil.com", "/"},
		{"absolute https", "https://evil.com/path", "/"},
		{"backslash becomes valid path", "\\evil.com", "/evil.com"}, // After normalization, becomes /evil.com
		{"double backslash", "\\\\evil.com", "/"},
		{"no leading slash", "evil.com", "/"},
		{"javascript scheme", "javascript:alert(1)", "/"},
		{"data scheme", "data:text/html,<h1>hi</h1>", "/"},
		{"protocol relative with path", "//evil.com/path", "/"},
		{"mixed backslash forward", "\\\\evil.com/path", "/"},
		{"with fragment", "/page#section", "/page"},
		{"with query and fragment", "/page?id=1#section", "/page?id=1"},
		{"path with dots", "/../../etc/passwd", "/../../etc/passwd"},          // Path traversal - allowed as relative
		{"encoded slashes get decoded", "/path%2Fto%2Fpage", "/path/to/page"}, // url.Parse decodes
		{"mailto scheme", "mailto:user@example.com", "/"},
		{"ftp scheme", "ftp://evil.com", "/"},
		{"triple slash", "///evil.com", "/"},
		{"newline injection", "/page\nLocation: evil.com", "/"}, // Contains newline
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeRedirectURL(tt.input)
			assert.Equal(t, tt.expected, result, "Input: %s", tt.input)
		})
	}
}
