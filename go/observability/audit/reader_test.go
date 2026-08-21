package audit_test

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/observability/audit"
)

func TestReader_ReadAll(t *testing.T) {
	t.Parallel()

	jsonl := `{"ts":"2026-08-21T10:00:00Z","actor":"user@example.com","action":"move","resource":"files/123"}
{"ts":"2026-08-21T10:01:00Z","actor":"user@example.com","action":"move","resource":"files/456"}

{"ts":"2026-08-21T10:02:00Z","actor":"admin@example.com","action":"trash","resource":"files/789"}
`
	entries, err := audit.ReadEntries(strings.NewReader(jsonl))
	if err != nil {
		t.Fatalf("ReadEntries failed: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].Actor != "user@example.com" || entries[0].Action != "move" || entries[0].Resource != "files/123" {
		t.Errorf("unexpected entry 0: %+v", entries[0])
	}
	if entries[2].Actor != "admin@example.com" || entries[2].Action != "trash" {
		t.Errorf("unexpected entry 2: %+v", entries[2])
	}
}

func TestReader_InvalidJSON(t *testing.T) {
	t.Parallel()

	jsonl := `{"ts":"2026-08-21T10:00:00Z","actor":"user@example.com"}
{invalid-json}
`
	r := audit.NewReader(strings.NewReader(jsonl))

	entry1, err := r.Read()
	if err != nil {
		t.Fatalf("first read failed: %v", err)
	}
	if entry1.Actor != "user@example.com" {
		t.Errorf("unexpected actor: %s", entry1.Actor)
	}

	_, err = r.Read()
	if !errors.Is(err, audit.ErrInvalidJSONLine) {
		t.Fatalf("expected ErrInvalidJSONLine, got: %v", err)
	}
}

const resourceRes1 = "res/1"

func TestFilter(t *testing.T) {
	t.Parallel()

	entries := []audit.Entry{
		{Actor: actorAlice, Action: actionCreate, Resource: resourceRes1},
		{Actor: actorBob, Action: actionCreate, Resource: "res/2"},
		{Actor: actorAlice, Action: actionDelete, Resource: "res/3"},
	}

	// Filter by actor.
	aliceEntries := audit.Filter(entries, func(e audit.Entry) bool {
		return e.Actor == actorAlice
	})
	if len(aliceEntries) != 2 {
		t.Errorf("expected 2 entries for alice, got %d", len(aliceEntries))
	}

	// Filter with nil predicate.
	all := audit.Filter(entries, nil)
	if len(all) != 3 {
		t.Errorf("expected 3 entries for nil predicate, got %d", len(all))
	}
}

func TestAggregateScorecard(t *testing.T) {
	t.Parallel()

	entries := []audit.Entry{
		{Actor: actorAlice, Action: actionCreate, Resource: resourceRes1},
		{Actor: actorBob, Action: actionCreate, Resource: resourceRes1},
		{Actor: actorAlice, Action: actionDelete, Resource: "res/2"},
		{Actor: actorAlice, Action: actionCreate, Resource: resourceRes1},
	}

	sc := audit.AggregateScorecard(entries)

	if sc.Total != 4 {
		t.Errorf("expected total=4, got %d", sc.Total)
	}
	if sc.ByActor[actorAlice] != 3 || sc.ByActor[actorBob] != 1 {
		t.Errorf("unexpected ByActor: %+v", sc.ByActor)
	}
	if sc.ByAction[actionCreate] != 3 || sc.ByAction[actionDelete] != 1 {
		t.Errorf("unexpected ByAction: %+v", sc.ByAction)
	}
	if len(sc.TopResources) == 0 || sc.TopResources[0] != resourceRes1 {
		t.Errorf("expected top resource %s, got: %v", resourceRes1, sc.TopResources)
	}
}

func FuzzAuditReader(f *testing.F) {
	f.Add([]byte(`{"ts":"2026-08-21T10:00:00Z","actor":"alice","action":"test","resource":"r1"}` + "\n"))
	f.Add([]byte(`{"ts":"","actor":"","action":"","resource":""}` + "\n"))
	f.Add([]byte("not-json\n"))
	f.Add([]byte("\n\n\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		r := audit.NewReader(bytes.NewReader(data))
		for {
			_, err := r.Read()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				// Corrupted lines should return ErrInvalidJSONLine or read errors.
				break
			}
		}
	})
}
