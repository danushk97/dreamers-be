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

// Phone removes non-digits, keeps max 10 digits.
func Phone(s string) string {
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
		return string(digits[:10])
	}
	return string(digits)
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

// MaxLen truncates to max length.
func MaxLen(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max])
}
