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
		{"mixed", "+91 9876543210", "9198765432"},
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
