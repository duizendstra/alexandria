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

func TestCallGeminiExtract_TransportErrorDoesNotLeakAPIKey(t *testing.T) {
	const apiKey = "AIzaSyFAKE-not-a-real-key-0000000"

	if strings.Contains(geminiEndpoint, "key=") {
		t.Fatalf("the default endpoint carries a key parameter: %q", geminiEndpoint)
	}

	// Started then closed: the refused connection is what produces the *url.Error
	// whose message embeds the request URL.
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close()

	restore := geminiEndpoint
	geminiEndpoint = srv.URL + "/v1beta/models/gemini-1.5-flash:generateContent"
	defer func() { geminiEndpoint = restore }()

	// The production entry point, so that reintroducing the key anywhere between
	// here and the request fails this test.
	_, err := callGeminiExtract(context.Background(), "worked on something", apiKey)
	if err == nil {
		t.Fatal("want a transport error from a closed listener, got nil")
	}
	if strings.Contains(err.Error(), apiKey) {
		t.Errorf("API key leaked into the error, which is logged and written to the response: %v", err)
	}

	// Control: the same failure with the key in the query string must leak it.
	// Without this arm the assertion above also passes when no key is sent at all.
	_, err = callGeminiExtractAt(context.Background(), geminiEndpoint+"?key="+apiKey, "worked on something", "")
	if err == nil {
		t.Fatal("want a transport error from a closed listener, got nil")
	}
	if !strings.Contains(err.Error(), apiKey) {
		t.Errorf("control did not leak the key, so this test cannot distinguish a fix from a no-op: %v", err)
	}
}

func TestCallGeminiExtract_SendsKeyAsHeader(t *testing.T) {
	const apiKey = "AIzaSyFAKE-not-a-real-key-0000000"

	var gotHeader, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Goog-Api-Key")
		gotQuery = r.URL.Query().Get("key")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	restore := geminiEndpoint
	geminiEndpoint = srv.URL
	defer func() { geminiEndpoint = restore }()

	// The call fails on the 500; what matters is how the key travelled.
	_, _ = callGeminiExtract(context.Background(), "worked on something", apiKey)

	if gotHeader != apiKey {
		t.Errorf("X-Goog-Api-Key header = %q, want the key", gotHeader)
	}
	if gotQuery != "" {
		t.Errorf("key travelled in the query string as %q", gotQuery)
	}
}
