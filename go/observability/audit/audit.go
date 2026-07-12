package audit

import (
	"context"
)

// ActorHeader is the HTTP header used to identify the caller.
const ActorHeader = "X-Dui-Actor"

// Entry represents a single audit log entry.
type Entry struct {
	Time     string `json:"ts"`
	Actor    string `json:"actor"`
	Action   string `json:"action"`
	Resource string `json:"resource"`
}

// Scorecard summarises audit activity by actor and action domain.
type Scorecard struct {
	Total        int            `json:"total"`
	ByActor      map[string]int `json:"by_actor"`
	ByAction     map[string]int `json:"by_action"`
	TopResources []string       `json:"top_resources,omitempty"`
}

// Writer appends audit entries to a log.
type Writer interface {
	Log(ctx context.Context, entry Entry) error
}
