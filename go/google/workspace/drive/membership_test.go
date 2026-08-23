package drive_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/google/workspace/drive"
)

const (
	fieldRole         = "role"
	fieldType         = "type"
	fieldEmailAddress = "emailAddress"
	permToDelete      = "perm-to-delete"
	existingWriter    = "existing-writer@example.com"
	existingOrganizer = "existing-organizer@example.com"
	newUser           = "new-user@example.com"
)

func TestRoleRank(t *testing.T) {
	if drive.RoleRank(drive.RoleOrganizer) <= drive.RoleRank(drive.RoleFileOrganizer) {
		t.Error("organizer should outrank fileOrganizer")
	}
	if drive.RoleRank(drive.RoleFileOrganizer) <= drive.RoleRank(drive.RoleWriter) {
		t.Error("fileOrganizer should outrank writer")
	}
	if drive.RoleRank(drive.RoleWriter) <= drive.RoleRank(drive.RoleCommenter) {
		t.Error("writer should outrank commenter")
	}
	if drive.RoleRank(drive.RoleCommenter) <= drive.RoleRank(drive.RoleReader) {
		t.Error("commenter should outrank reader")
	}
	if drive.RoleRank("unknown_role") != 0 {
		t.Errorf("unknown role should rank 0, got %d", drive.RoleRank("unknown_role"))
	}
}

func TestService_EnsureDriveMembership(t *testing.T) {
	ctx := context.Background()

	var (
		createdPerms []string
		updatedPerms []string
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/permissions"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"permissions": []map[string]any{
					{
						"id":              "perm-existing-writer",
						fieldRole:         "writer",
						fieldType:         "user",
						fieldEmailAddress: existingWriter,
					},
					{
						"id":              "perm-existing-organizer",
						fieldRole:         "organizer",
						fieldType:         "user",
						fieldEmailAddress: existingOrganizer,
					},
				},
			})
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/permissions"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			email, _ := body[fieldEmailAddress].(string)
			createdPerms = append(createdPerms, email)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":              "perm-new",
				fieldRole:         body[fieldRole],
				fieldType:         body[fieldType],
				fieldEmailAddress: email,
			})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/permissions/perm-existing-writer"):
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			updatedPerms = append(updatedPerms, "perm-existing-writer")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":      "perm-existing-writer",
				fieldRole: body[fieldRole],
			})
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/permissions/"+permToDelete):
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer ts.Close()

	svc := newRetryingService(t, ctx, ts.URL)

	// Scenario 1: New Member Creation.
	resNew, err := svc.EnsureDriveMembership(ctx, "drive-1", newUser, drive.RoleWriter, false)
	if err != nil {
		t.Fatalf("EnsureDriveMembership (new): %v", err)
	}
	if resNew.Action != drive.MembershipActionCreated {
		t.Errorf("expected CREATED, got %s", resNew.Action)
	}
	if len(createdPerms) != 1 || createdPerms[0] != newUser {
		t.Errorf("expected created perm for new user, got %v", createdPerms)
	}

	// Scenario 2: Upgrade Existing Member (writer -> organizer).
	resUpgrade, err := svc.EnsureDriveMembership(ctx, "drive-1", existingWriter, drive.RoleOrganizer, false)
	if err != nil {
		t.Fatalf("EnsureDriveMembership (upgrade): %v", err)
	}
	if resUpgrade.Action != drive.MembershipActionUpgraded {
		t.Errorf("expected UPGRADED, got %s", resUpgrade.Action)
	}
	if resUpgrade.PreviousRole != drive.RoleWriter {
		t.Errorf("expected previous role 'writer', got %s", resUpgrade.PreviousRole)
	}
	if len(updatedPerms) != 1 {
		t.Errorf("expected 1 updated perm, got %v", updatedPerms)
	}

	// Scenario 3: Unchanged Member (organizer requested for organizer).
	resUnchanged, err := svc.EnsureDriveMembership(ctx, "drive-1", existingOrganizer, drive.RoleOrganizer, false)
	if err != nil {
		t.Fatalf("EnsureDriveMembership (unchanged): %v", err)
	}
	if resUnchanged.Action != drive.MembershipActionUnchanged {
		t.Errorf("expected UNCHANGED, got %s", resUnchanged.Action)
	}

	// Scenario 4: Dry-Run Mode (should not mutate).
	resDryRun, err := svc.EnsureDriveMembership(ctx, "drive-1", "dryrun-user@example.com", drive.RoleOrganizer, true)
	if err != nil {
		t.Fatalf("EnsureDriveMembership (dryrun): %v", err)
	}
	if resDryRun.Action != drive.MembershipActionCreated {
		t.Errorf("expected CREATED action in dry-run, got %s", resDryRun.Action)
	}
	if !resDryRun.DryRun {
		t.Error("expected DryRun=true in result")
	}

	// Scenario 5: Delete Member.
	if err := svc.DeleteDriveMember(ctx, "drive-1", permToDelete); err != nil {
		t.Fatalf("DeleteDriveMember: %v", err)
	}

	// Scenario 6: Delete File Permission.
	if err := svc.DeleteFilePermission(ctx, "file-1", permToDelete); err != nil {
		t.Fatalf("DeleteFilePermission: %v", err)
	}

	// Scenario 7: List File Permissions.
	filePerms, err := svc.ListFilePermissions(ctx, "file-1")
	if err != nil {
		t.Fatalf("ListFilePermissions: %v", err)
	}
	if len(filePerms) != 2 {
		t.Errorf("expected 2 file permissions, got %d", len(filePerms))
	}
}

func FuzzRoleRank(f *testing.F) {
	f.Add("reader")
	f.Add("commenter")
	f.Add("writer")
	f.Add("fileOrganizer")
	f.Add("organizer")
	f.Add("invalid")
	f.Add("")

	f.Fuzz(func(t *testing.T, role string) {
		rank := drive.RoleRank(role)
		switch role {
		case drive.RoleReader:
			if rank != 10 {
				t.Errorf("expected 10 for reader, got %d", rank)
			}
		case drive.RoleCommenter:
			if rank != 20 {
				t.Errorf("expected 20 for commenter, got %d", rank)
			}
		case drive.RoleWriter:
			if rank != 30 {
				t.Errorf("expected 30 for writer, got %d", rank)
			}
		case drive.RoleFileOrganizer:
			if rank != 40 {
				t.Errorf("expected 40 for fileOrganizer, got %d", rank)
			}
		case drive.RoleOrganizer:
			if rank != 50 {
				t.Errorf("expected 50 for organizer, got %d", rank)
			}
		default:
			if rank != 0 {
				t.Errorf("expected 0 for unknown role %q, got %d", role, rank)
			}
		}
	})
}
