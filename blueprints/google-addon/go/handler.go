package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/duizendstra/alexandria/go/platform/web"
)

// AddOnEvent represents a Google Workspace Add-ons HTTP trigger event payload.
type AddOnEvent struct {
	Calendar          map[string]any `json:"calendar,omitempty"`
	CommonEventObject struct {
		UserLocale string `json:"userLocale,omitempty"`
		HostApp    string `json:"hostApp,omitempty"`
		Platform   string `json:"platform,omitempty"`
		FormInputs map[string]struct {
			StringInputs struct {
				Value []string `json:"value,omitempty"`
			} `json:"stringInputs,omitempty"`
		} `json:"formInputs,omitempty"`
		Parameters map[string]string `json:"parameters,omitempty"`
	} `json:"commonEventObject,omitempty"`
	AuthorizationEventObject struct {
		UserOAuthToken string `json:"userOAuthToken,omitempty"`
	} `json:"authorizationEventObject,omitempty"`
}

// FormInputValue extracts the first string input from formInputs.
func (e *AddOnEvent) FormInputValue(fieldName string) string {
	if e.CommonEventObject.FormInputs == nil {
		return ""
	}
	input, ok := e.CommonEventObject.FormInputs[fieldName]
	if !ok || len(input.StringInputs.Value) == 0 {
		return ""
	}
	return input.StringInputs.Value[0]
}

// CardResponse represents Google Workspace Add-ons Card JSON v2 response.
type CardResponse struct {
	RenderActions struct {
		Action struct {
			Navigations []NavigationAction `json:"navigations,omitempty"`
		} `json:"action,omitempty"`
	} `json:"renderActions,omitempty"`
}

// NavigationAction defines card navigation (push, update, or pop).
type NavigationAction struct {
	PushCard   *CardV2 `json:"pushCard,omitempty"`
	UpdateCard *CardV2 `json:"updateCard,omitempty"`
}

// CardV2 represents a single Card UI component.
type CardV2 struct {
	Header   *CardHeader   `json:"header,omitempty"`
	Sections []CardSection `json:"sections,omitempty"`
}

// CardHeader defines the title and icon of a Card.
type CardHeader struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle,omitempty"`
	ImageURL string `json:"imageUrl,omitempty"`
}

// CardSection groups related widgets in a Card.
type CardSection struct {
	Header      string   `json:"header,omitempty"`
	Collapsible bool     `json:"collapsible,omitempty"`
	Widgets     []Widget `json:"widgets"`
}

// Widget represents an individual UI widget inside a card section.
type Widget struct {
	TextParagraph *TextParagraphWidget `json:"textParagraph,omitempty"`
	TextInput     *TextInputWidget     `json:"textInput,omitempty"`
	ButtonList    *ButtonListWidget    `json:"buttonList,omitempty"`
}

// TextParagraphWidget displays styled text.
type TextParagraphWidget struct {
	Text string `json:"text"`
}

// TextInputWidget displays an editable text input box.
type TextInputWidget struct {
	Name      string `json:"name"`
	Label     string `json:"label"`
	Value     string `json:"value,omitempty"`
	Multiline bool   `json:"multiline,omitempty"`
	HintText  string `json:"hintText,omitempty"`
}

// ButtonListWidget contains a row of clickable action buttons.
type ButtonListWidget struct {
	Buttons []Button `json:"buttons"`
}

// Button defines an interactive card button.
type Button struct {
	Text    string       `json:"text"`
	OnClick ButtonAction `json:"onClick"`
}

// ButtonAction specifies the action handler endpoint or action function.
type ButtonAction struct {
	Action *ActionSpec `json:"action,omitempty"`
}

// ActionSpec specifies the action name and parameters to send back.
type ActionSpec struct {
	FunctionName string            `json:"functionName"`
	Parameters   map[string]string `json:"parameters,omitempty"`
}

// HandleCalendarTrigger routes incoming Google Workspace HTTP requests.
func HandleCalendarTrigger(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	event, err := web.DecodeJSON[AddOnEvent](w, r, web.DefaultMaxJSONSize)
	if err != nil {
		slog.WarnContext(ctx, "failed to decode add-on event", slog.Any("err", err))
		web.WriteError(w, err)
		return
	}

	action := event.CommonEventObject.Parameters["action"]
	slog.InfoContext(ctx, "processing add-on trigger",
		slog.String("hostApp", event.CommonEventObject.HostApp),
		slog.String("action", action),
	)

	switch action {
	case "analyze":
		handleAnalyzeNote(ctx, w, event)
	default:
		// Default: Return Homepage Card
		handleHomepage(w)
	}
}

// handleHomepage returns the initial sidebar UI card.
func handleHomepage(w http.ResponseWriter) {
	var resp CardResponse
	homepageCard := &CardV2{
		Header: &CardHeader{
			Title:    "AI Time Tracker (Go)",
			Subtitle: "Powered by Alexandria & Gemini Flash",
			ImageURL: "https://www.gstatic.com/images/icons/material/system/1x/schedule_black_24dp.png",
		},
		Sections: []CardSection{
			{
				Widgets: []Widget{
					{
						TextParagraph: &TextParagraphWidget{
							Text: "📝 <b>Describe your work in plain English:</b><br>Our Go backend will structure your time log with Gemini AI.",
						},
					},
					{
						TextInput: &TextInputWidget{
							Name:      "work_note",
							Label:     "Work Note",
							Multiline: true,
							HintText:  "e.g. Spent 2 hours reviewing PRs and deploying Cloud Run for Acme",
						},
					},
					{
						ButtonList: &ButtonListWidget{
							Buttons: []Button{
								{
									Text: "✨ Analyze & Structure with AI",
									OnClick: ButtonAction{
										Action: &ActionSpec{
											FunctionName: "onAnalyzeNote",
											Parameters: map[string]string{
												"action": "analyze",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	resp.RenderActions.Action.Navigations = []NavigationAction{
		{PushCard: homepageCard},
	}

	web.EncodeJSON(w, http.StatusOK, resp)
}

// handleAnalyzeNote calls Gemini AI with structured schema and returns confirmation card.
func handleAnalyzeNote(ctx context.Context, w http.ResponseWriter, event AddOnEvent) {
	note := event.FormInputValue("work_note")
	if note == "" {
		handleHomepage(w)
		return
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		slog.ErrorContext(ctx, "GEMINI_API_KEY environment variable is not configured")
		http.Error(w, "GEMINI_API_KEY not configured", http.StatusInternalServerError)
		return
	}

	entry, err := callGeminiExtract(ctx, note, apiKey)
	if err != nil {
		slog.ErrorContext(ctx, "failed to extract time entry with Gemini", slog.Any("err", err))
		http.Error(w, fmt.Sprintf("AI extraction failed: %v", err), http.StatusInternalServerError)
		return
	}
	entry.Sanitize()

	var resp CardResponse
	confirmCard := &CardV2{
		Header: &CardHeader{
			Title:    "Confirm Time Entry",
			Subtitle: entry.Title,
		},
		Sections: []CardSection{
			{
				Widgets: []Widget{
					{
						TextInput: &TextInputWidget{
							Name:  "entry_title",
							Label: "Event Title",
							Value: entry.Title,
						},
					},
					{
						TextInput: &TextInputWidget{
							Name:  "entry_duration",
							Label: "Duration (Hours)",
							Value: fmt.Sprintf("%.2f", entry.DurationHours),
						},
					},
					{
						TextInput: &TextInputWidget{
							Name:      "entry_summary",
							Label:     "Description / Notes",
							Multiline: true,
							Value:     entry.Summary,
						},
					},
					{
						TextParagraph: &TextParagraphWidget{
							Text: fmt.Sprintf("Client: <b>%s</b> | Project: <b>%s</b>", entry.Client, entry.Project),
						},
					},
				},
			},
		},
	}

	resp.RenderActions.Action.Navigations = []NavigationAction{
		{PushCard: confirmCard},
	}

	web.EncodeJSON(w, http.StatusOK, resp)
}

// callGeminiExtract calls Google AI Studio Gemini API with JSON response format.
// The key travels in a header, never in this URL. A transport failure yields a
// *url.Error carrying the whole URL, and callGeminiExtractAt's errors are both
// logged and written into the HTTP response.
// A var, not a const, so a test can exercise callGeminiExtract itself rather than
// only the helper beneath it.
//
//nolint:gochecknoglobals // Test seam: lets the key-leak regression test drive callGeminiExtract itself.
var geminiEndpoint = "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent"

func callGeminiExtract(ctx context.Context, note, apiKey string) (*TimeEntry, error) {
	return callGeminiExtractAt(ctx, geminiEndpoint, note, apiKey)
}

// callGeminiExtractAt takes the endpoint so a test can point it at a closed
// listener and assert on the resulting error.
func callGeminiExtractAt(ctx context.Context, endpoint, note, apiKey string) (*TimeEntry, error) {
	prompt := fmt.Sprintf(`You are an expert time-tracking assistant.
Analyze this work note and extract:
- client (string)
- project (string)
- duration_hours (float, e.g. 0.5, 1.0, 1.5, default to 1.0)
- title (string, e.g. "[Client] Task")
- summary (string, 1-2 sentence timesheet description)

Work note: %q`, note)

	reqPayload := map[string]any{
		"contents": []any{
			map[string]any{
				"parts": []any{
					map[string]any{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature":      0.2,
			"responseMimeType": "application/json",
		},
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", apiKey)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gemini request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("failed to decode gemini response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from gemini")
	}

	rawJSON := geminiResp.Candidates[0].Content.Parts[0].Text
	var entry TimeEntry
	if err := json.Unmarshal([]byte(rawJSON), &entry); err != nil {
		return nil, fmt.Errorf("failed to parse structured time entry JSON: %w", err)
	}

	return &entry, nil
}
