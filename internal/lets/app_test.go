package lets

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
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

func TestCalendarEventLinks(t *testing.T) {
	dates := []WednesdayDate{
		{
			Value:             "2026-06-17",
			Label:             "Wednesday, Jun 17, 2026",
			Reserved:          true,
			Contact:           "Pat",
			SuggestedLocation: "Tacos",
		},
	}

	event := calendarEventForDate("2026-06-17", dates)
	if event == nil {
		t.Fatal("expected calendar event")
	}
	if event.Start.Hour() != 12 || event.End.Hour() != 13 {
		t.Fatalf("event = %s to %s, want noon to 1pm", event.Start, event.End)
	}

	google, err := url.Parse(event.GoogleCalendarLink())
	if err != nil {
		t.Fatalf("parse google link: %v", err)
	}
	if google.Host != "www.google.com" {
		t.Fatalf("google host = %q", google.Host)
	}
	if got := google.Query().Get("text"); got != "Lunch" {
		t.Fatalf("google text = %q", got)
	}
	if got := google.Query().Get("location"); got != "Tacos" {
		t.Fatalf("google location = %q", got)
	}

	outlook, err := url.Parse(event.OutlookCalendarLink())
	if err != nil {
		t.Fatalf("parse outlook link: %v", err)
	}
	if outlook.Host != "outlook.live.com" {
		t.Fatalf("outlook host = %q", outlook.Host)
	}
	if got := outlook.Query().Get("subject"); got != "Lunch" {
		t.Fatalf("outlook subject = %q", got)
	}
	if got := outlook.Query().Get("location"); got != "Tacos" {
		t.Fatalf("outlook location = %q", got)
	}
}

func TestReserveRedirectShowsCalendarLinks(t *testing.T) {
	db, err := OpenSQLite(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	app, err := NewApp(db)
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	app.now = func() time.Time {
		return time.Date(2026, time.June, 15, 10, 30, 0, 0, time.Local)
	}

	form := url.Values{}
	form.Set("date", "2026-06-17")
	form.Set("contact", "pat@example.com")
	form.Set("suggested_location", "Tacos")
	req := httptest.NewRequest(http.MethodPost, "/requests", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "saved=1") || !strings.Contains(location, "date=2026-06-17") {
		t.Fatalf("redirect location = %q", location)
	}

	req = httptest.NewRequest(http.MethodGet, location, nil)
	rec = httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Saved. See you then.",
		"https://www.google.com/calendar/render?",
		"https://outlook.live.com/calendar/0/deeplink/compose?",
		"Reserved by pat@example.com",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body does not contain %q:\n%s", want, body)
		}
	}
}
