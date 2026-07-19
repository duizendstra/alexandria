package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ActorHeader is the HTTP header used to identify the caller.
const ActorHeader = "X-Dui-Actor"

// Entry represents a single audit log entry.
//
// Time is owned by the Writer implementation: Log stamps it with the writer's
// clock, overwriting any caller-supplied value. On the wire it is encoded as
// an RFC 3339 string under the "ts" key (see MarshalJSON), so existing JSONL
// log files keep parsing unchanged.
type Entry struct {
	Time     time.Time
	Actor    string
	Action   string
	Resource string
}

// entryJSON is the stable wire representation of Entry. Field names and order
// must not change: existing audit log files depend on them.
type entryJSON struct {
	Time     string `json:"ts"`
	Actor    string `json:"actor"`
	Action   string `json:"action"`
	Resource string `json:"resource"`
}

// MarshalJSON encodes the entry using the stable wire format: the timestamp
// is written under the "ts" key as an RFC 3339 string. A zero Time is encoded
// as an empty string, matching the historical format for unstamped entries.
func (e Entry) MarshalJSON() ([]byte, error) {
	ts := ""
	if !e.Time.IsZero() {
		ts = e.Time.Format(time.RFC3339)
	}

	data, err := json.Marshal(entryJSON{
		Time:     ts,
		Actor:    e.Actor,
		Action:   e.Action,
		Resource: e.Resource,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal audit entry: %w", err)
	}

	return data, nil
}

// UnmarshalJSON decodes the stable wire format written by MarshalJSON,
// parsing the "ts" key as an RFC 3339 timestamp. An empty or absent "ts"
// yields a zero Time; a malformed timestamp is an error.
func (e *Entry) UnmarshalJSON(data []byte) error {
	var raw entryJSON
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("unmarshal audit entry: %w", err)
	}

	var ts time.Time
	if raw.Time != "" {
		parsed, err := time.Parse(time.RFC3339, raw.Time)
		if err != nil {
			return fmt.Errorf("parse audit entry time %q: %w", raw.Time, err)
		}
		ts = parsed
	}

	e.Time = ts
	e.Actor = raw.Actor
	e.Action = raw.Action
	e.Resource = raw.Resource

	return nil
}

// Scorecard summarises audit activity by actor and action domain.
type Scorecard struct {
	Total        int            `json:"total"`
	ByActor      map[string]int `json:"by_actor"`
	ByAction     map[string]int `json:"by_action"`
	TopResources []string       `json:"top_resources,omitempty"`
}

// Writer appends audit entries to a log.
//
// Implementations own the Entry.Time field: Log must stamp it with the
// implementation's clock, overwriting any caller-supplied value.
type Writer interface {
	Log(ctx context.Context, entry Entry) error
}
