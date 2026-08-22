package drive_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/duizendstra/alexandria/go/google/workspace/drive"
	googledrive "google.golang.org/api/drive/v3"
)

const (
	fieldDrives     = "drives"
	fieldName       = "name"
	fieldFiles      = "files"
	fileToMoveID    = "file-to-move"
	sharedDriveName = "Agora - Team Drive"
)

func TestService_SharedDriveOperations(t *testing.T) {
	ctx := context.Background()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/drives"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "drive-new-123",
				fieldName: body[fieldName],
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/drives/drive-123"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "drive-123",
				fieldName: "Target Shared Drive",
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/drives"):
			q := r.URL.Query().Get("q")
			pageToken := r.URL.Query().Get("pageToken")
			if strings.Contains(q, "ExactMatch") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					fieldDrives: []map[string]any{
						{"id": "drive-exact", fieldName: "ExactMatch"},
					},
				})

				return
			}
			if strings.Contains(q, "NonExistent") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					fieldDrives: []map[string]any{},
				})

				return
			}

			if pageToken == "" {
				_ = json.NewEncoder(w).Encode(map[string]any{
					fieldDrives: []map[string]any{
						{"id": "dr-1", fieldName: "Drive 1"},
					},
					"nextPageToken": "page-2",
				})
			} else {
				_ = json.NewEncoder(w).Encode(map[string]any{
					fieldDrives: []map[string]any{
						{"id": "dr-2", fieldName: "Drive 2"},
					},
				})
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	svc := newRetryingService(t, ctx, ts.URL)

	// Test CreateSharedDrive.
	created, err := svc.CreateSharedDrive(ctx, sharedDriveName, "req-123",
		drive.WithDriveRestrictions(&googledrive.DriveRestrictions{
			DomainUsersOnly: true,
		}))
	if err != nil {
		t.Fatalf("CreateSharedDrive: %v", err)
	}
	if created.Id != "drive-new-123" {
		t.Errorf("unexpected drive ID: %s", created.Id)
	}

	// Test FindSharedDriveByName (Found).
	found, err := svc.FindSharedDriveByName(ctx, "ExactMatch")
	if err != nil {
		t.Fatalf("FindSharedDriveByName: %v", err)
	}
	if found == nil || found.Id != "drive-exact" {
		t.Errorf("expected drive-exact, got %v", found)
	}

	// Test FindSharedDriveByName (Not Found).
	missing, err := svc.FindSharedDriveByName(ctx, "NonExistent")
	if err == nil || !errors.Is(err, drive.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing drive, got %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for missing drive, got %v", missing)
	}

	// Test GetSharedDrive.
	got, err := svc.GetSharedDrive(ctx, "drive-123")
	if err != nil {
		t.Fatalf("GetSharedDrive: %v", err)
	}
	if got.Name != "Target Shared Drive" {
		t.Errorf("expected 'Target Shared Drive', got %s", got.Name)
	}

	// Test ListSharedDrives (Pagination).
	allDrives, err := svc.ListSharedDrives(ctx, 10)
	if err != nil {
		t.Fatalf("ListSharedDrives: %v", err)
	}
	if len(allDrives) != 2 {
		t.Fatalf("expected 2 paginated drives, got %d", len(allDrives))
	}
}

func TestService_FolderAndFileOperations(t *testing.T) {
	ctx := context.Background()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/files"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       "folder-new-1",
				fieldName:  body[fieldName],
				"mimeType": "application/vnd.google-apps.folder",
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/files/"+fileToMoveID):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      fileToMoveID,
				"parents": []string{"parent-auto-1", "parent-auto-2"},
			})
		case r.Method == http.MethodPatch && strings.HasSuffix(r.URL.Path, "/files/"+fileToMoveID):
			addParents := r.URL.Query().Get("addParents")
			removeParents := r.URL.Query().Get("removeParents")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      fileToMoveID,
				"parents": []string{addParents},
				fieldName: "moved-file.pdf",
			})
			if !strings.Contains(removeParents, "parent-auto-1") && !strings.Contains(removeParents, "parent-old") {
				t.Errorf("unexpected removeParents: %s", removeParents)
			}
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/files/file-props"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": "file-props",
			})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/files/file-trash"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "file-trash",
				"trashed": true,
			})
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files"):
			q := r.URL.Query().Get("q")
			if strings.Contains(q, "MyFolder") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					fieldFiles: []map[string]any{
						{"id": "folder-found-id", fieldName: "MyFolder"},
					},
				})

				return
			}
			if strings.Contains(q, "prop_key") {
				_ = json.NewEncoder(w).Encode(map[string]any{
					fieldFiles: []map[string]any{
						{"id": "folder-prop-id", fieldName: "PropFolder"},
					},
				})

				return
			}

			_ = json.NewEncoder(w).Encode(map[string]any{
				fieldFiles: []map[string]any{},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	svc := newRetryingService(t, ctx, ts.URL)

	// Test CreateFolder.
	f, err := svc.CreateFolder(ctx, "parent-1", "Archive 2026", map[string]string{"migrated": "true"})
	if err != nil {
		t.Fatalf("CreateFolder: %v", err)
	}
	if f.ID != "folder-new-1" {
		t.Errorf("unexpected folder id: %s", f.ID)
	}

	// Test FindFolder.
	fid, err := svc.FindFolder(ctx, "parent-1", "MyFolder")
	if err != nil {
		t.Fatalf("FindFolder: %v", err)
	}
	if fid != "folder-found-id" {
		t.Errorf("expected folder-found-id, got %s", fid)
	}

	// Test FindFolder (Absent).
	missingFID, err := svc.FindFolder(ctx, "parent-1", "AbsentFolder")
	if err != nil {
		t.Fatalf("FindFolder absent: %v", err)
	}
	if missingFID != "" {
		t.Errorf("expected empty string for absent folder, got %s", missingFID)
	}

	// Test FindFolderByProperty.
	propFID, err := svc.FindFolderByProperty(ctx, "prop_key", "prop_val")
	if err != nil {
		t.Fatalf("FindFolderByProperty: %v", err)
	}
	if propFID != "folder-prop-id" {
		t.Errorf("expected folder-prop-id, got %s", propFID)
	}

	// Test MoveFile with explicit oldParentID.
	moved, err := svc.MoveFile(ctx, fileToMoveID, "parent-old", "parent-new")
	if err != nil {
		t.Fatalf("MoveFile explicit: %v", err)
	}
	if moved.Id != fileToMoveID {
		t.Errorf("unexpected moved file ID: %s", moved.Id)
	}

	// Test MoveFile with auto-discovered oldParentID (empty oldParentID).
	movedAuto, err := svc.MoveFile(ctx, fileToMoveID, "", "parent-new")
	if err != nil {
		t.Fatalf("MoveFile auto-discovered: %v", err)
	}
	if movedAuto.Id != fileToMoveID {
		t.Errorf("unexpected moved file ID: %s", movedAuto.Id)
	}

	// Test SetAppProperties.
	if err := svc.SetAppProperties(ctx, "file-props", map[string]string{"k": "v"}); err != nil {
		t.Fatalf("SetAppProperties: %v", err)
	}

	// Test TrashFile.
	trashed, err := svc.TrashFile(ctx, "file-trash")
	if err != nil {
		t.Fatalf("TrashFile: %v", err)
	}
	if !trashed {
		t.Errorf("expected trashed=true, got false")
	}
}

// TestEscapeQueryValue proves the helper doubles backslashes before escaping
// quotes, so every input reads back as a literal inside a '...' query term.
func TestEscapeQueryValue(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain", in: "Quarterly Report", want: "Quarterly Report"},
		{name: "single quote", in: "Valentine's Day", want: `Valentine\'s Day`},
		{name: "trailing backslash", in: `archive\`, want: `archive\\`},
		{name: "backslash quote", in: `it\'s`, want: `it\\\'s`},
		{name: "empty", in: "", want: ""},
		{name: "unicode", in: "Résumé – 日本語 'ok'", want: `Résumé – 日本語 \'ok\'`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := drive.EscapeQueryValue(tc.in); got != tc.want {
				t.Errorf("EscapeQueryValue(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestService_QueryValuesAreEscaped proves the public lookups send a "q"
// parameter whose string literals are fully escaped — a trailing backslash or
// a backslash-quote in a caller-supplied value must not change the query.
func TestService_QueryValuesAreEscaped(t *testing.T) {
	ctx := context.Background()

	var mu sync.Mutex
	var queries []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		queries = append(queries, r.URL.Query().Get("q"))
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{fieldDrives: []any{}, fieldFiles: []any{}})
	}))
	defer ts.Close()

	svc := newRetryingService(t, ctx, ts.URL)

	const hostile = `it\'s a trap\`

	if _, err := svc.FindSharedDriveByName(ctx, hostile); err != nil && !errors.Is(err, drive.ErrNotFound) {
		t.Fatalf("FindSharedDriveByName: %v", err)
	}
	if _, err := svc.FindFolder(ctx, "parent-1", hostile); err != nil {
		t.Fatalf("FindFolder: %v", err)
	}
	if _, err := svc.FindFolderByProperty(ctx, hostile, hostile); err != nil {
		t.Fatalf("FindFolderByProperty: %v", err)
	}

	const escaped = `it\\\'s a trap\\`
	want := []string{
		`name = '` + escaped + `'`,
		`name = '` + escaped + `' and mimeType = 'application/vnd.google-apps.folder' and 'parent-1' in parents and trashed = false`,
		`appProperties has { key='` + escaped + `' and value='` + escaped + `' } and mimeType = 'application/vnd.google-apps.folder' and trashed = false`,
	}

	mu.Lock()
	defer mu.Unlock()
	if len(queries) != len(want) {
		t.Fatalf("expected %d queries, got %d: %q", len(want), len(queries), queries)
	}
	for i := range want {
		if queries[i] != want[i] {
			t.Errorf("query %d:\n got %s\nwant %s", i, queries[i], want[i])
		}
	}
}
