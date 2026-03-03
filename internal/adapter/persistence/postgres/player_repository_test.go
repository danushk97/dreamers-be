package postgres

import (
	"testing"
	"time"
)

func TestAgeConditions(t *testing.T) {
	now := time.Date(2025, 3, 3, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		gender    string
		ageFilter string
		wantLen   int
	}{
		{"MALE", "below-30", 1},
		{"MALE", "31-40", 2},
		{"MALE", "41-50", 2},
		{"MALE", "50+", 1},
		{"FEMALE", "above-30", 1},
		{"FEMALE", "31-40", 0},
		{"MALE", "all", 0},
	}
	for _, tt := range tests {
		t.Run(tt.gender+"_"+tt.ageFilter, func(t *testing.T) {
			conds := ageConditions(tt.gender, tt.ageFilter, now)
			if len(conds) != tt.wantLen {
				t.Errorf("ageConditions() = %d conditions, want %d", len(conds), tt.wantLen)
			}
		})
	}
}

func TestPlaceholder(t *testing.T) {
	if placeholder(1) != "1" {
		t.Error("placeholder(1)")
	}
	if placeholder(15) != "15" {
		t.Error("placeholder(15)")
	}
}
