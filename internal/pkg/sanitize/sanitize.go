package sanitize

import (
	"regexp"
	"strings"
	"unicode"
)

// String trims and collapses whitespace, removes control characters.
func String(s string) string {
	s = strings.TrimSpace(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return collapseSpace(b.String())
}

func collapseSpace(s string) string {
	return regexp.MustCompile(`\s+`).ReplaceAllString(strings.TrimSpace(s), " ")
}

// Phone trims leading/trailing spaces, strips +91 prefix if present, then extracts
// up to 10 digits. Returns the normalized 10-digit number for storage/validation.
func Phone(s string) string {
	s = strings.TrimSpace(s)
	// Trim +91 prefix if present
	if strings.HasPrefix(s, "+91") {
		s = strings.TrimSpace(s[3:])
	}
	var digits []rune
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
			if len(digits) >= 10 {
				break
			}
		}
	}
	if len(digits) > 10 {
		return strings.ToLower(string(digits[:10]))
	}
	return strings.ToLower(string(digits))
}

// AlphanumericID strips to alphanumeric and hyphen/underscore only.
func AlphanumericID(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// URL validates and sanitizes URL strings (basic safety).
func URL(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// Only allow http/https
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return ""
	}
	return s
}

// OneOf returns s if it's in allowed, else "".
func OneOf(s string, allowed []string) string {
	s = strings.TrimSpace(s)
	for _, a := range allowed {
		if a == s {
			return s
		}
	}
	return ""
}

// TNBAID trims the "TNBA/" prefix if present (case-insensitive), validates format
// "number/number" (e.g. 7/0515), and returns the normalized value for storage.
// Returns "" if invalid.
func TNBAID(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Trim TNBA/ prefix (case-insensitive)
	if strings.HasPrefix(strings.ToUpper(s), "TNBA/") {
		s = strings.TrimSpace(s[5:])
	}
	// Validate format: digits/digits (e.g. 7/0515)
	if !regexp.MustCompile(`^\d+/\d+$`).MatchString(s) {
		return ""
	}
	return strings.ToLower(s)
}

// MaxLen truncates to max length.
func MaxLen(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
