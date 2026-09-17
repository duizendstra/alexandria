package main

// TimeEntry represents a structured time-tracking entry parsed by Gemini AI.
type TimeEntry struct {
	Client        string  `json:"client"`
	Project       string  `json:"project"`
	DurationHours float64 `json:"duration_hours"`
	Title         string  `json:"title"`
	Summary       string  `json:"summary"`
}

// Sanitize ensures fields have reasonable defaults.
func (t *TimeEntry) Sanitize() {
	if t.Client == "" {
		t.Client = "General"
	}
	if t.Project == "" {
		t.Project = "Work"
	}
	if t.DurationHours <= 0 {
		t.DurationHours = 1.0
	}
	if t.Title == "" {
		t.Title = "[" + t.Client + "] " + t.Project
	}
}
