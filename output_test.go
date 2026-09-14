package main

import (
	"testing"
	"time"
)

func TestFriendlyTime(t *testing.T) {
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.Local)
	tests := []struct {
		t    time.Time
		want string
	}{
		{time.Date(2026, 9, 14, 9, 5, 0, 0, time.Local), "today 09:05"},
		{time.Date(2026, 9, 11, 18, 30, 0, 0, time.Local), "Fri 18:30"},
		{time.Date(2026, 3, 2, 8, 0, 0, 0, time.Local), "Mar 02"},
		{time.Date(2025, 12, 31, 8, 0, 0, 0, time.Local), "2025-12-31"},
	}
	for _, tt := range tests {
		if got := friendlyTime(tt.t, now); got != tt.want {
			t.Errorf("friendlyTime(%v) = %q, want %q", tt.t, got, tt.want)
		}
	}
}

func TestHumanSize(t *testing.T) {
	for n, want := range map[int64]string{0: "0 B", 1023: "1023 B", 1536: "1.5 KB", 5 << 20: "5.0 MB"} {
		if got := humanSize(n); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", n, got, want)
		}
	}
}
