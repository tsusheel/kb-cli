package cmd

import (
	"testing"
	"time"

	"github.com/spf13/viper"
)

func TestParseDate(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	tests := []struct {
		input    string
		expected time.Time
		hasError bool
	}{
		{"", time.Time{}, false},
		{"today", today, false},
		{"tomorrow", today.AddDate(0, 0, 1), false},
		{"tmrw", today.AddDate(0, 0, 1), false},
		{"yesterday", today.AddDate(0, 0, -1), false},
		{"+3d", today.AddDate(0, 0, 3), false},
		{"+1w", today.AddDate(0, 0, 7), false},
		{"in 2 days", today.AddDate(0, 0, 2), false},
		{"2026-09-10", time.Date(2026, 9, 10, 0, 0, 0, 0, now.Location()), false},
		{"2026-09-10 14:30", time.Date(2026, 9, 10, 14, 30, 0, 0, now.Location()), false},
		{"invalid-date-string-xyz", time.Time{}, true},
	}

	for _, tc := range tests {
		got, err := ParseDate(tc.input)
		if tc.hasError {
			if err == nil {
				t.Errorf("ParseDate(%q) expected error, got nil", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseDate(%q) unexpected error: %v", tc.input, err)
			}
			if !got.Equal(tc.expected) {
				t.Errorf("ParseDate(%q) = %v, expected %v", tc.input, got, tc.expected)
			}
		}
	}
}

func TestParseDateWithConfig(t *testing.T) {
	viper.Set("date_format", "02/01/2006")
	defer viper.Set("date_format", "")

	now := time.Now()
	got, err := ParseDate("25/12/2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := time.Date(2026, 12, 25, 0, 0, 0, 0, now.Location())
	if !got.Equal(expected) {
		t.Errorf("got %v, expected %v", got, expected)
	}
}
