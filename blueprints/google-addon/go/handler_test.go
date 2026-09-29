package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
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
	geminiEndpoint = srv.URL + "/v1beta/models/gemini-3.8-flash:generateContent"
	defer func() { geminiEndpoint = restore }()

	// The production entry point, so that reintroducing the key anywhere between
	// here and the request fails this test.
	_, err := callGeminiExtract(context.Background(), "worked on something", apiKey)
	if err == nil {
		t.Fatal("want a transport error from a closed listener, got nil")
	}
	if strings.Contains(err.Error(), apiKey) {
		t.Errorf("API key leaked into the error, which is logged: %v", err)
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

// captureLogs routes the default logger into a buffer for the rest of the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return &buf
}

func analyzeRequest(t *testing.T) *http.Request {
	t.Helper()
	body := `{"commonEventObject": {"hostApp": "CALENDAR",
		"parameters": {"action": "analyze"},
		"formInputs": {"work_note": {"stringInputs": {"value": ["2h on the workshop"]}}}}}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	return req
}

func TestHandleAnalyzeNote_UpstreamErrorStaysInTheLog(t *testing.T) {
	const marker = "UPSTREAM-DETAIL-7f3a"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"` + marker + `"}}`))
	}))
	defer srv.Close()

	restore := geminiEndpoint
	geminiEndpoint = srv.URL
	defer func() { geminiEndpoint = restore }()
	t.Setenv("GEMINI_API_KEY", "AIzaSyFAKE-not-a-real-key-0000000")
	logs := captureLogs(t)

	rec := httptest.NewRecorder()
	HandleCalendarTrigger(rec, analyzeRequest(t))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 so the sidebar renders the card", rec.Code)
	}
	var resp CardResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not a card response: %v\n%s", err, rec.Body.String())
	}
	if len(resp.RenderActions.Action.Navigations) != 1 || resp.RenderActions.Action.Navigations[0].PushCard == nil {
		t.Fatalf("want one pushCard navigation, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Gemini could not analyze the note") {
		t.Errorf("response lacks the short message: %s", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), marker) {
		t.Errorf("upstream body reached the response: %s", rec.Body.String())
	}
	// Control: the detail is not lost, it is in the log.
	if !strings.Contains(logs.String(), marker) {
		t.Errorf("upstream body missing from the log, so the test cannot tell it was dropped from the response on purpose:\n%s", logs.String())
	}
}

func TestHandleAnalyzeNote_MissingKeyIsACard(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "")
	_ = captureLogs(t)

	rec := httptest.NewRecorder()
	HandleCalendarTrigger(rec, analyzeRequest(t))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"pushCard"`) || !strings.Contains(rec.Body.String(), "no Gemini API key") {
		t.Errorf("want an error card naming the missing key, got %s", rec.Body.String())
	}
}

func TestHandleHomepage_ButtonCallsTheServiceURL(t *testing.T) {
	body := `{"commonEventObject": {"hostApp": "CALENDAR"}}`
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "https://tracker-abc.a.run.app/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	HandleCalendarTrigger(rec, req)

	got := rec.Body.String()
	if !strings.Contains(got, `"function":"https://tracker-abc.a.run.app"`) {
		t.Errorf("button action does not call the service URL: %s", got)
	}
	if !strings.Contains(got, `"parameters":[{"key":"action","value":"analyze"}]`) {
		t.Errorf("button parameters are not a key/value list: %s", got)
	}
	if strings.Contains(got, "functionName") {
		t.Errorf("functionName is an Apps Script field, not an HTTP one: %s", got)
	}
}
