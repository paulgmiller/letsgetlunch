package lets

import (
	"strings"
	"testing"
	"time"
)

func TestWednesdaysBetweenIncludesTodayWhenWednesday(t *testing.T) {
	start := time.Date(2026, time.June, 17, 10, 30, 0, 0, time.UTC)
	end := start.AddDate(0, 2, 0)

	dates := wednesdaysBetween(start, end)
	if len(dates) == 0 {
		t.Fatal("expected at least one Wednesday")
	}
	if dates[0].Value != "2026-06-17" {
		t.Fatalf("first date = %q, want 2026-06-17", dates[0].Value)
	}
}

func TestWednesdaysBetweenStartsWithNextWednesday(t *testing.T) {
	start := time.Date(2026, time.June, 15, 10, 30, 0, 0, time.UTC)
	end := start.AddDate(0, 2, 0)

	dates := wednesdaysBetween(start, end)
	if len(dates) == 0 {
		t.Fatal("expected at least one Wednesday")
	}
	if dates[0].Value != "2026-06-17" {
		t.Fatalf("first date = %q, want 2026-06-17", dates[0].Value)
	}
	for _, date := range dates {
		parsed, err := time.Parse("2006-01-02", date.Value)
		if err != nil {
			t.Fatalf("parse %q: %v", date.Value, err)
		}
		if parsed.Weekday() != time.Wednesday {
			t.Fatalf("%s is %s, want Wednesday", date.Value, parsed.Weekday())
		}
		if parsed.After(end) {
			t.Fatalf("%s is after end date %s", date.Value, end.Format("2006-01-02"))
		}
	}
}

func TestValidateSubmission(t *testing.T) {
	dates := []WednesdayDate{
		{Value: "2026-06-17", Label: "Wednesday, Jun 17, 2026"},
	}

	tests := []struct {
		name         string
		dates        []WednesdayDate
		contact      string
		selectedDate string
		wantErr      string
	}{
		{
			name:         "valid",
			contact:      "pat@example.com",
			selectedDate: "2026-06-17",
		},
		{
			name:         "missing contact",
			selectedDate: "2026-06-17",
			wantErr:      "name or email",
		},
		{
			name:    "missing date",
			contact: "Pat",
			wantErr: "choose a Wednesday",
		},
		{
			name:         "date outside allowed list",
			contact:      "Pat",
			selectedDate: "2026-06-24",
			wantErr:      "available Wednesdays",
		},
		{
			name: "reserved date",
			dates: []WednesdayDate{
				{Value: "2026-06-17", Label: "Wednesday, Jun 17, 2026", Reserved: true},
			},
			contact:      "Pat",
			selectedDate: "2026-06-17",
			wantErr:      "already reserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDates := dates
			if tt.dates != nil {
				testDates = tt.dates
			}
			_, err := validateSubmission(tt.contact, tt.selectedDate, testDates)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("validateSubmission returned error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.wantErr)
			}
		})
	}
}
