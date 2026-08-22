package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTimeEntry_Sanitize(t *testing.T) {
	entry := TimeEntry{
		Client:        "",
		Project:       "",
		DurationHours: 0,
		Title:         "",
	}
	entry.Sanitize()

	if entry.Client != "General" {
		t.Errorf("expected client General, got %s", entry.Client)
	}
	if entry.Project != "Work" {
		t.Errorf("expected project Work, got %s", entry.Project)
	}
	if entry.DurationHours != 1.0 {
		t.Errorf("expected duration 1.0, got %f", entry.DurationHours)
	}
	if entry.Title != "[General] Work" {
		t.Errorf("expected title [General] Work, got %s", entry.Title)
	}
}

func TestHandleCalendarTrigger_MethodNotAllowed(t *testing.T) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	HandleCalendarTrigger(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rec.Code)
	}
}

func TestHandleCalendarTrigger_Homepage(t *testing.T) {
	body := `{"commonEventObject": {"hostApp": "CALENDAR"}}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	HandleCalendarTrigger(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	resBody := rec.Body.String()
	if !strings.Contains(resBody, "AI Time Tracker (Go)") {
		t.Errorf("expected response to contain homepage title, got %s", resBody)
	}
}
