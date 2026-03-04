package sanitize

import (
	"testing"
)

func TestString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"trim", "  foo  ", "foo"},
		{"collapse", "foo   bar\tbaz", "foo bar baz"},
		{"control", "foo\x00bar\nbaz", "foobar baz"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := String(tt.input); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPhone(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"digits only", "9876543210", "9876543210"},
		{"with dashes", "987-654-3210", "9876543210"},
		{"too long", "12345678901", "1234567890"},
		{"plus91 with space", "+91 9876543210", "9876543210"},
		{"plus91 no space", "+919876543210", "9876543210"},
		{"leading trailing spaces", "  9876543210  ", "9876543210"},
		{"plus91 and spaces", "  +91 9876543210  ", "9876543210"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Phone(tt.input); got != tt.want {
				t.Errorf("Phone() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOneOf(t *testing.T) {
	allowed := []string{"MALE", "FEMALE"}
	if got := OneOf("MALE", allowed); got != "MALE" {
		t.Errorf("OneOf(MALE) = %q", got)
	}
	if got := OneOf("male", allowed); got != "" {
		t.Errorf("OneOf(male) = %q, want empty", got)
	}
	if got := OneOf("  MALE  ", allowed); got != "MALE" {
		t.Errorf("OneOf(trimmed) = %q", got)
	}
}

func TestMaxLen(t *testing.T) {
	if got := MaxLen("hello", 3); got != "hel" {
		t.Errorf("MaxLen() = %q", got)
	}
	if got := MaxLen("hi", 10); got != "hi" {
		t.Errorf("MaxLen(short) = %q", got)
	}
}

func TestURL(t *testing.T) {
	if got := URL("https://example.com/file"); got != "https://example.com/file" {
		t.Errorf("URL(https) = %q", got)
	}
	if got := URL("javascript:alert(1)"); got != "" {
		t.Errorf("URL(js) = %q, want empty", got)
	}
}

func TestTNBAID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"with prefix", "TNBA/7/0515", "7/0515"},
		{"lowercase prefix", "tnba/7/0515", "7/0515"},
		{"no prefix", "7/0515", "7/0515"},
		{"leading trailing spaces", "  7/0515  ", "7/0515"},
		{"spaces with prefix", "  TNBA/7/0515  ", "7/0515"},
		{"multi digit", "12/1234", "12/1234"},
		{"empty", "", ""},
		{"invalid no slash", "TNBA123", ""},
		{"invalid letters", "7/ABC", ""},
		{"invalid format", "7-0515", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TNBAID(tt.input); got != tt.want {
				t.Errorf("TNBAID(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
