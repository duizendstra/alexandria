package sheets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	mockSpreadsheetID = "test-sheet-id"
	mockFolderID      = "folder-xyz-123"
	mockTabTitle      = "ExistingTab"
	mockKeepTitle     = "KeepMe"
	mockStaleTitle    = "OldStaleTab"
	mockSyncedTitle   = "Synced Doc"
)

type mockSheetsHandler struct {
	mu                sync.Mutex
	getCalls          int
	batchUpdates      int
	receivedBatchReqs []*sheets.Request
	clears            int
	valUpdates        []string
	createdTitle      string
	movedFileID       string
	parentAdded       string
	sheets            []*sheets.Sheet
	deletedSheetIDs   []int64
	failDeleteSheet   bool
}

func writeJSON(w http.ResponseWriter, val any) {
	w.Header().Set("Content-Type", "application/json")
	b, err := json.Marshal(val)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
	_, _ = w.Write(b)
}

func (m *mockSheetsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := r.URL.Path

	if len(m.sheets) == 0 {
		m.sheets = []*sheets.Sheet{
			{
				Properties: &sheets.SheetProperties{
					SheetId: 0,
					Title:   mockTabTitle,
					GridProperties: &sheets.GridProperties{
						RowCount:    50,
						ColumnCount: 10,
					},
				},
			},
		}
	}

	// Drive files update (move folder).
	if strings.Contains(path, "/files/") {
		m.movedFileID = path[strings.LastIndex(path, "/")+1:]
		m.parentAdded = r.URL.Query().Get("addParents")
		writeJSON(w, &drive.File{Id: m.movedFileID})

		return
	}

	// Spreadsheets Create.
	if path == "/v4/spreadsheets" && r.Method == http.MethodPost {
		var req sheets.Spreadsheet
		_ = json.NewDecoder(r.Body).Decode(&req)
		m.createdTitle = req.Properties.Title

		respSheets := make([]*sheets.Sheet, len(req.Sheets))
		for i, s := range req.Sheets {
			title := DefaultSheetTitle
			if s.Properties != nil && s.Properties.Title != "" {
				title = s.Properties.Title
			}
			respSheets[i] = &sheets.Sheet{
				Properties: &sheets.SheetProperties{
					SheetId: int64(100 + i),
					Title:   title,
					GridProperties: &sheets.GridProperties{
						RowCount:    100,
						ColumnCount: 26,
					},
				},
			}
		}
		m.sheets = respSheets

		writeJSON(w, &sheets.Spreadsheet{
			SpreadsheetId:  mockSpreadsheetID,
			SpreadsheetUrl: "https://docs.google.com/spreadsheets/d/" + mockSpreadsheetID,
			Properties:     req.Properties,
			Sheets:         respSheets,
		})

		return
	}

	// Spreadsheets Get.
	if strings.HasPrefix(path, "/v4/spreadsheets/") && r.Method == http.MethodGet {
		m.getCalls++
		sp := &sheets.Spreadsheet{
			SpreadsheetId: mockSpreadsheetID,
			Properties: &sheets.SpreadsheetProperties{
				Title: "Test Spreadsheet",
			},
			Sheets: m.sheets,
		}
		writeJSON(w, sp)

		return
	}

	// Spreadsheets BatchUpdate.
	if strings.HasSuffix(path, ":batchUpdate") && r.Method == http.MethodPost {
		m.batchUpdates++
		var bReq sheets.BatchUpdateSpreadsheetRequest
		_ = json.NewDecoder(r.Body).Decode(&bReq)
		m.receivedBatchReqs = append(m.receivedBatchReqs, bReq.Requests...)

		for _, req := range bReq.Requests {
			if req.DeleteSheet == nil {
				continue
			}
			if m.failDeleteSheet {
				http.Error(w, "delete sheet rejected", http.StatusBadRequest)

				return
			}
			m.deletedSheetIDs = append(m.deletedSheetIDs, req.DeleteSheet.SheetId)
		}

		newSheet := &sheets.Sheet{
			Properties: &sheets.SheetProperties{
				SheetId: 999,
				Title:   "NewTab",
				GridProperties: &sheets.GridProperties{
					RowCount:    100,
					ColumnCount: 26,
				},
			},
		}
		m.sheets = append(m.sheets, newSheet)
		resp := &sheets.BatchUpdateSpreadsheetResponse{
			SpreadsheetId: mockSpreadsheetID,
			Replies: []*sheets.Response{
				{
					AddSheet: &sheets.AddSheetResponse{
						Properties: newSheet.Properties,
					},
				},
			},
		}
		writeJSON(w, resp)

		return
	}

	// Values Clear.
	if strings.HasSuffix(path, ":clear") && r.Method == http.MethodPost {
		m.clears++
		writeJSON(w, &sheets.ClearValuesResponse{
			SpreadsheetId: mockSpreadsheetID,
		})

		return
	}

	// Values Update.
	if strings.Contains(path, "/values/") && r.Method == http.MethodPut {
		m.valUpdates = append(m.valUpdates, path)
		writeJSON(w, &sheets.UpdateValuesResponse{
			SpreadsheetId: mockSpreadsheetID,
		})

		return
	}

	http.NotFound(w, r)
}

func setupTestService(t *testing.T) (*Service, *mockSheetsHandler) {
	t.Helper()

	handler := &mockSheetsHandler{}
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	ctx := context.Background()
	sheetsClient, err := sheets.NewService(ctx, option.WithEndpoint(srv.URL), option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("failed to create sheets service: %v", err)
	}

	driveClient, err := drive.NewService(ctx, option.WithEndpoint(srv.URL), option.WithoutAuthentication())
	if err != nil {
		t.Fatalf("failed to create drive service: %v", err)
	}

	svc := NewWithClients(sheetsClient, driveClient, nil)

	return svc, handler
}

func TestReplaceTab_Existing(t *testing.T) {
	svc, handler := setupTestService(t)
	ctx := context.Background()

	tbl := NewTable("Col1", "Col2")
	tbl.AddRowValues("Val1", "Val2")

	res, err := svc.ReplaceTab(ctx, mockSpreadsheetID, TabSpec{
		Title: mockTabTitle,
		Data:  tbl,
		Theme: ThemeCorporateNavy(),
	})
	if err != nil {
		t.Fatalf("ReplaceTab failed: %v", err)
	}

	if res.Title != mockTabTitle {
		t.Errorf("expected title %s, got %s", mockTabTitle, res.Title)
	}
	if res.RowsWritten != 1 {
		t.Errorf("expected 1 row written, got %d", res.RowsWritten)
	}
	if handler.clears != 1 {
		t.Errorf("expected 1 clear call, got %d", handler.clears)
	}
	if len(handler.valUpdates) == 0 {
		t.Errorf("expected value updates to be recorded")
	}
}

func TestReplaceTab_MissingWithCreate(t *testing.T) {
	svc, handler := setupTestService(t)
	ctx := context.Background()

	tbl := NewTable("HeaderA")
	tbl.AddRowValues("Row1")

	res, err := svc.ReplaceTab(ctx, mockSpreadsheetID, TabSpec{
		Title:           "NewTab",
		CreateIfMissing: true,
		Data:            tbl,
	})
	if err != nil {
		t.Fatalf("ReplaceTab failed: %v", err)
	}

	if res.SheetID != 999 {
		t.Errorf("expected sheet ID 999 from mock reply, got %d", res.SheetID)
	}
	if handler.batchUpdates < 1 {
		t.Errorf("expected batchUpdate for AddSheet")
	}
}

func TestCreateSpreadsheet_WithDriveRelocation(t *testing.T) {
	svc, handler := setupTestService(t)
	ctx := context.Background()

	tbl := NewTable("Name", "Role")
	tbl.AddRowValues("Alice", "Admin")

	spec := DocumentSpec{
		Title:    "Quarterly Export",
		Locale:   "nl_NL",
		FolderID: mockFolderID,
		Tabs: []TabSpec{
			{
				Title: "Staff",
				Data:  tbl,
				Theme: ThemeCorporateNavy(),
			},
		},
	}

	res, err := svc.CreateSpreadsheet(ctx, spec)
	if err != nil {
		t.Fatalf("CreateSpreadsheet failed: %v", err)
	}

	if res.SpreadsheetID != mockSpreadsheetID {
		t.Errorf("expected %s, got %s", mockSpreadsheetID, res.SpreadsheetID)
	}
	if handler.createdTitle != "Quarterly Export" {
		t.Errorf("expected created title 'Quarterly Export', got %q", handler.createdTitle)
	}
	if handler.parentAdded != mockFolderID {
		t.Errorf("expected parent added %s, got %q", mockFolderID, handler.parentAdded)
	}
}

func TestSyncDocument_WithPruning(t *testing.T) {
	svc, handler := setupTestService(t)
	ctx := context.Background()

	// Populate initial sheets in mock.
	handler.sheets = []*sheets.Sheet{
		{
			Properties: &sheets.SheetProperties{
				SheetId: 1,
				Title:   mockKeepTitle,
			},
		},
		{
			Properties: &sheets.SheetProperties{
				SheetId: 2,
				Title:   mockStaleTitle,
			},
		},
	}

	tbl := NewTable("Header1")
	tbl.AddRowValues("Data1")

	spec := DocumentSpec{
		Title:     mockSyncedTitle,
		PruneTabs: true,
		Tabs: []TabSpec{
			{
				Title: mockKeepTitle,
				Data:  tbl,
			},
		},
	}

	res, err := svc.SyncDocument(ctx, mockSpreadsheetID, spec)
	if err != nil {
		t.Fatalf("SyncDocument failed: %v", err)
	}

	if len(res.Tabs) != 1 {
		t.Fatalf("expected 1 tab in result, got %d", len(res.Tabs))
	}
	if handler.batchUpdates < 1 {
		t.Errorf("expected batchUpdates to execute deletion/formatting")
	}
	if len(handler.deletedSheetIDs) != 1 || handler.deletedSheetIDs[0] != 2 {
		t.Errorf("expected only the stale tab (sheet 2) to be deleted, got %v", handler.deletedSheetIDs)
	}
}

func TestSyncDocument_PruneKeepsGIDAddressedTab(t *testing.T) {
	svc, handler := setupTestService(t)
	ctx := context.Background()

	handler.sheets = []*sheets.Sheet{
		{
			Properties: &sheets.SheetProperties{
				SheetId: 1,
				Title:   "Report",
			},
		},
		{
			Properties: &sheets.SheetProperties{
				SheetId: 2,
				Title:   mockStaleTitle,
			},
		},
	}

	tbl := NewTable("Header1")
	tbl.AddRowValues("Data1")

	spec := DocumentSpec{
		Title:     mockSyncedTitle,
		PruneTabs: true,
		Tabs: []TabSpec{
			{
				// Addressed by GID; the title differs from the existing tab's title.
				GID:   1,
				Title: "Renamed",
				Data:  tbl,
			},
		},
	}

	res, err := svc.SyncDocument(ctx, mockSpreadsheetID, spec)
	if err != nil {
		t.Fatalf("SyncDocument failed: %v", err)
	}

	if len(res.Tabs) != 1 || res.Tabs[0].SheetID != 1 {
		t.Fatalf("expected the GID-addressed tab (sheet 1) to be synced, got %+v", res.Tabs)
	}
	for _, id := range handler.deletedSheetIDs {
		if id == 1 {
			t.Errorf("the tab synced by GID in this call must not be pruned, deleted %v", handler.deletedSheetIDs)
		}
	}
	if len(handler.deletedSheetIDs) != 1 || handler.deletedSheetIDs[0] != 2 {
		t.Errorf("expected only the stale tab (sheet 2) to be deleted, got %v", handler.deletedSheetIDs)
	}
}

func TestSyncDocument_PruneDeleteErrorSurfaces(t *testing.T) {
	svc, handler := setupTestService(t)
	ctx := context.Background()

	handler.sheets = []*sheets.Sheet{
		{
			Properties: &sheets.SheetProperties{
				SheetId: 1,
				Title:   mockKeepTitle,
			},
		},
		{
			Properties: &sheets.SheetProperties{
				SheetId: 2,
				Title:   mockStaleTitle,
			},
		},
	}
	handler.failDeleteSheet = true

	tbl := NewTable("Header1")
	tbl.AddRowValues("Data1")

	spec := DocumentSpec{
		Title:     mockSyncedTitle,
		PruneTabs: true,
		Tabs: []TabSpec{
			{
				Title: mockKeepTitle,
				Data:  tbl,
			},
		},
	}

	res, err := svc.SyncDocument(ctx, mockSpreadsheetID, spec)
	if err == nil {
		t.Fatalf("expected SyncDocument to surface the failed prune, got result %+v", res)
	}
	if !strings.Contains(err.Error(), "prune stale tabs") {
		t.Errorf("expected prune context in error, got %q", err.Error())
	}
}

func TestReplaceTab_SkipFormatting(t *testing.T) {
	svc, handler := setupTestService(t)
	ctx := context.Background()

	tbl := NewTable("Col1", "Col2")
	tbl.AddRowValues("Val1", "Val2")

	res, err := svc.ReplaceTab(ctx, mockSpreadsheetID, TabSpec{
		Title:          mockTabTitle,
		SkipFormatting: true,
		Data:           tbl,
	})
	if err != nil {
		t.Fatalf("ReplaceTab failed: %v", err)
	}

	if res.Title != mockTabTitle {
		t.Errorf("expected title %s, got %s", mockTabTitle, res.Title)
	}
	if handler.clears != 1 {
		t.Errorf("expected 1 clear call, got %d", handler.clears)
	}
	if len(handler.valUpdates) == 0 {
		t.Errorf("expected value updates to be recorded")
	}

	// Verify contract: with SkipFormatting=true, zero RepeatCell, AddBanding, or UpdateSheetProperties requests.
	for _, req := range handler.receivedBatchReqs {
		if req.RepeatCell != nil {
			t.Errorf("expected zero RepeatCell requests under SkipFormatting, got %+v", req.RepeatCell)
		}
		if req.AddBanding != nil {
			t.Errorf("expected zero AddBanding requests under SkipFormatting, got %+v", req.AddBanding)
		}
		if req.UpdateSheetProperties != nil {
			t.Errorf("expected zero UpdateSheetProperties requests under SkipFormatting, got %+v", req.UpdateSheetProperties)
		}
	}
}

func TestReplaceTab_RichLinks(t *testing.T) {
	svc, handler := setupTestService(t)
	ctx := context.Background()

	tbl := NewTable("Name", "Link")
	tbl.AddRow(Text("Alice"), Hyperlink("https://example.com/alice", "Alice Profile"))

	res, err := svc.ReplaceTab(ctx, mockSpreadsheetID, TabSpec{
		Title: mockTabTitle,
		Data:  tbl,
		Theme: ThemeCorporateNavy(),
	})
	if err != nil {
		t.Fatalf("ReplaceTab failed: %v", err)
	}

	if res.RowsWritten != 1 {
		t.Errorf("expected 1 row written, got %d", res.RowsWritten)
	}

	var lastFormatIndex = -1
	var firstRichLinkIndex = -1

	for idx, req := range handler.receivedBatchReqs {
		if req.RepeatCell != nil || req.AddBanding != nil || req.UpdateDimensionProperties != nil || req.UpdateSheetProperties != nil {
			lastFormatIndex = idx
		}
		if req.UpdateCells != nil && req.UpdateCells.Fields == fieldMaskTextFormatRuns {
			if firstRichLinkIndex == -1 {
				firstRichLinkIndex = idx
			}
		}
	}

	if firstRichLinkIndex == -1 {
		t.Fatalf("expected UpdateCells with textFormatRuns for rich link")
	}

	if lastFormatIndex != -1 && firstRichLinkIndex <= lastFormatIndex {
		t.Errorf("contract violation: rich links (index %d) must be applied AFTER formatting requests (last format index %d)",
			firstRichLinkIndex, lastFormatIndex)
	}
}
