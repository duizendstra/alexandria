package sheets

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/duizendstra/alexandria/go/retry/gcp"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	defaultMinRows  = int64(10)
	defaultMinCols  = int64(10)
	extraRowsBuffer = int64(5)
	extraColsBuffer = int64(2)
)

var (
	// ErrMissingSpreadsheetID indicates that an empty spreadsheet ID was provided.
	ErrMissingSpreadsheetID = errors.New("sheets: spreadsheetID is required")

	// ErrMissingTabTitleOrGID indicates that neither a Title nor a valid GID was provided.
	ErrMissingTabTitleOrGID = errors.New("sheets: tab Title or GID is required")

	// ErrMissingTabs indicates that DocumentSpec contains no tabs.
	ErrMissingTabs = errors.New("sheets: at least one tab is required in DocumentSpec")

	// ErrTabNotFound indicates that the requested tab could not be found.
	ErrTabNotFound = errors.New("sheets: tab not found")
)

// Config holds configuration parameters for the Sheets service.
type Config struct {
	Logger *slog.Logger
}

// Service provides high-level operations on Google Spreadsheets and Sheets.
type Service struct {
	sheets *sheets.Service
	drive  *drive.Service
	log    *slog.Logger
}

// New creates a new Service instance using the provided client options.
func New(ctx context.Context, cfg Config, clientOpts ...option.ClientOption) (*Service, error) {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}

	ss, err := sheets.NewService(ctx, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("sheets: create sheets service: %w", err)
	}

	// Drive service is optional for folder relocation.
	ds, _ := drive.NewService(ctx, clientOpts...)

	return &Service{
		sheets: ss,
		drive:  ds,
		log:    log,
	}, nil
}

// NewWithClients creates a Service directly from pre-constructed API clients.
func NewWithClients(sheetsSvc *sheets.Service, driveSvc *drive.Service, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}

	return &Service{
		sheets: sheetsSvc,
		drive:  driveSvc,
		log:    log,
	}
}

// RawService returns the underlying Google Sheets API client.
func (s *Service) RawService() *sheets.Service {
	return s.sheets
}

// RawDriveService returns the underlying Google Drive API client (if initialized).
func (s *Service) RawDriveService() *drive.Service {
	return s.drive
}

// ReplaceTab idempotently synchronizes a single tab:
// 1. Resolves the tab by GID or Title (creating if missing when CreateIfMissing=true).
// 2. Expands grid dimensions to accommodate the new dataset.
// 3. Clears old data in the tab.
// 4. Writes headers and data using safe dual-mode partitioning (RAW for literals, USER_ENTERED for formulas).
// 5. Applies theme styling (frozen panes, colored header, zebra banding, column widths).
func (s *Service) ReplaceTab(ctx context.Context, spreadsheetID string, spec TabSpec) (*TabResult, error) {
	if spreadsheetID == "" {
		return nil, ErrMissingSpreadsheetID
	}
	if spec.Title == "" && spec.GID < 0 {
		return nil, ErrMissingTabTitleOrGID
	}

	tab, err := s.resolveOrCreateTab(ctx, spreadsheetID, spec)
	if err != nil {
		return nil, err
	}

	targetRowCount, targetColCount, err := s.resizeTab(ctx, spreadsheetID, tab.Properties, spec)
	if err != nil {
		return nil, err
	}

	if err := s.clearTab(ctx, spreadsheetID, tab.Properties.Title); err != nil {
		return nil, err
	}

	rowsWritten, err := s.writeTabBatches(ctx, spreadsheetID, tab.Properties.Title, spec.Data)
	if err != nil {
		return nil, err
	}

	s.applyRichLinks(ctx, spreadsheetID, tab.Properties.SheetId, spec.Data)

	if !spec.SkipFormatting {
		var bandedIDs []int64
		for _, br := range tab.BandedRanges {
			if br != nil && br.BandedRangeId != 0 {
				bandedIDs = append(bandedIDs, br.BandedRangeId)
			}
		}

		s.applyTabFormatting(ctx, spreadsheetID, tab.Properties.SheetId, bandedIDs, spec)
	}

	s.log.InfoContext(ctx, "sheet tab synchronized",
		slog.String("spreadsheet_id", spreadsheetID),
		slog.String("tab_title", tab.Properties.Title),
		slog.Int64("gid", tab.Properties.SheetId),
		slog.Int("rows_written", rowsWritten),
	)

	return &TabResult{
		SheetID:     tab.Properties.SheetId,
		Title:       tab.Properties.Title,
		RowCount:    targetRowCount,
		ColumnCount: targetColCount,
		RowsWritten: rowsWritten,
	}, nil
}

// CreateSpreadsheet creates a new Google Spreadsheet with the given tabs,
// applies formatting, writes data, and optionally relocates it to a target Drive folder.
//
//nolint:gocritic // spec by value is intentional for caller immutability
func (s *Service) CreateSpreadsheet(ctx context.Context, spec DocumentSpec) (*DocumentResult, error) {
	if len(spec.Tabs) == 0 {
		return nil, ErrMissingTabs
	}

	title := spec.Title
	if title == "" {
		title = "Export"
	}
	locale := spec.Locale
	if locale == "" {
		locale = "nl_NL"
	}

	sheetDefs := buildInitialSheets(spec.Tabs)

	var created *sheets.Spreadsheet
	err := gcp.WithRetry(ctx, func() error {
		var e error
		created, e = s.sheets.Spreadsheets.Create(&sheets.Spreadsheet{
			Properties: &sheets.SpreadsheetProperties{
				Title:  title,
				Locale: locale,
			},
			Sheets: sheetDefs,
		}).Context(ctx).Do()
		if e != nil {
			return fmt.Errorf("sheets api create: %w", e)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sheets: create spreadsheet %q: %w", title, err)
	}

	tabResults, err := s.populateCreatedTabs(ctx, created, spec.Tabs)
	if err != nil {
		return nil, err
	}

	s.relocateToFolder(ctx, created.SpreadsheetId, spec.FolderID)

	sheetURL := created.SpreadsheetUrl
	if sheetURL == "" {
		sheetURL = "https://docs.google.com/spreadsheets/d/" + created.SpreadsheetId
	}

	return &DocumentResult{
		SpreadsheetID:  created.SpreadsheetId,
		SpreadsheetURL: sheetURL,
		Title:          title,
		Tabs:           tabResults,
	}, nil
}

// SyncDocument updates an existing spreadsheet (or creates a new one if spreadsheetID is empty).
// When PruneTabs is true, it removes any old tabs not listed in spec.Tabs.
//
//nolint:gocritic // spec by value is intentional for caller immutability
func (s *Service) SyncDocument(ctx context.Context, spreadsheetID string, spec DocumentSpec) (*DocumentResult, error) {
	if spreadsheetID == "" {
		return s.CreateSpreadsheet(ctx, spec)
	}

	var sp *sheets.Spreadsheet
	err := gcp.WithRetry(ctx, func() error {
		var e error
		sp, e = s.sheets.Spreadsheets.Get(spreadsheetID).
			Fields("spreadsheetId,properties.title,sheets.properties").
			Context(ctx).
			Do()
		if e != nil {
			return fmt.Errorf("sheets api get: %w", e)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sheets: get spreadsheet %s: %w", spreadsheetID, err)
	}

	tabResults := make([]TabResult, 0, len(spec.Tabs))
	for _, tabSpec := range spec.Tabs {
		tabSpec.CreateIfMissing = true
		res, err := s.ReplaceTab(ctx, spreadsheetID, tabSpec)
		if err != nil {
			return nil, err
		}
		tabResults = append(tabResults, *res)
	}

	if spec.PruneTabs {
		s.pruneStaleTabs(ctx, spreadsheetID, sp.Sheets, spec.Tabs)
	}

	return &DocumentResult{
		SpreadsheetID:  spreadsheetID,
		SpreadsheetURL: "https://docs.google.com/spreadsheets/d/" + spreadsheetID,
		Title:          sp.Properties.Title,
		Tabs:           tabResults,
	}, nil
}

func (s *Service) resolveOrCreateTab(ctx context.Context, spreadsheetID string, spec TabSpec) (*sheets.Sheet, error) {
	var sp *sheets.Spreadsheet
	err := gcp.WithRetry(ctx, func() error {
		var e error
		sp, e = s.sheets.Spreadsheets.Get(spreadsheetID).
			Fields("properties.title,sheets.properties,sheets.bandedRanges").
			Context(ctx).
			Do()
		if e != nil {
			return fmt.Errorf("sheets api get: %w", e)
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sheets: get spreadsheet %s: %w", spreadsheetID, err)
	}

	if tab := findMatchingSheet(sp.Sheets, spec); tab != nil {
		return tab, nil
	}

	if !spec.CreateIfMissing {
		return nil, fmt.Errorf("%w: %q (gid=%d) in spreadsheet %q", ErrTabNotFound, spec.Title, spec.GID, sp.Properties.Title)
	}

	return s.createNewTab(ctx, spreadsheetID, spec.Title)
}

func findMatchingSheet(existingSheets []*sheets.Sheet, spec TabSpec) *sheets.Sheet {
	for _, sheet := range existingSheets {
		if sheet.Properties == nil {
			continue
		}
		if spec.Title != "" && sheet.Properties.Title == spec.Title {
			return sheet
		}
		if spec.Title == "" && spec.GID >= 0 && sheet.Properties.SheetId == spec.GID {
			return sheet
		}
		if spec.GID > 0 && sheet.Properties.SheetId == spec.GID {
			return sheet
		}
	}

	return nil
}

func (s *Service) createNewTab(ctx context.Context, spreadsheetID, title string) (*sheets.Sheet, error) {
	if title == "" {
		title = DefaultSheetTitle
	}

	var tab *sheets.Sheet
	err := gcp.WithRetry(ctx, func() error {
		resp, e := s.sheets.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{{
				AddSheet: &sheets.AddSheetRequest{
					Properties: &sheets.SheetProperties{Title: title},
				},
			}},
		}).Context(ctx).Do()
		if e != nil {
			return fmt.Errorf("sheets: batch update add sheet: %w", e)
		}
		if len(resp.Replies) > 0 && resp.Replies[0].AddSheet != nil {
			tab = &sheets.Sheet{Properties: resp.Replies[0].AddSheet.Properties}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("sheets: add tab %q: %w", title, err)
	}

	return tab, nil
}

//nolint:gocritic // returns rowCount, colCount, err
func (s *Service) resizeTab(
	ctx context.Context,
	spreadsheetID string,
	tab *sheets.SheetProperties,
	spec TabSpec,
) (int64, int64, error) {
	rowsCount, colsCount, targetRowCount, targetColCount := calculateTargetGridDimensions(tab, spec.Data)

	if spec.SkipFormatting {
		return s.resizeTabGridOnly(ctx, spreadsheetID, tab, rowsCount, colsCount, targetRowCount, targetColCount)
	}

	frozenRows := spec.FrozenRows
	if frozenRows == 0 && spec.Data != nil && len(spec.Data.Headers) > 0 {
		frozenRows = 1
	}

	retryErr := gcp.WithRetry(ctx, func() error {
		_, e := s.sheets.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{{
				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Properties: &sheets.SheetProperties{
						SheetId: tab.SheetId,
						GridProperties: &sheets.GridProperties{
							RowCount:          targetRowCount,
							ColumnCount:       targetColCount,
							FrozenRowCount:    frozenRows,
							FrozenColumnCount: spec.FrozenCols,
						},
					},
					Fields: "gridProperties.rowCount,gridProperties.columnCount,gridProperties.frozenRowCount,gridProperties.frozenColumnCount",
				},
			}},
		}).Context(ctx).Do()
		if e != nil {
			return fmt.Errorf("sheets api resize: %w", e)
		}

		return nil
	})
	if retryErr != nil {
		return 0, 0, fmt.Errorf("sheets: resize/freeze tab %q: %w", tab.Title, retryErr)
	}

	return targetRowCount, targetColCount, nil
}

//nolint:gocritic // returns (rowsCount, colsCount, targetRowCount, targetColCount)
func calculateTargetGridDimensions(tab *sheets.SheetProperties, data *Table) (int64, int64, int64, int64) {
	rowsCount, colsCount := defaultMinRows, defaultMinCols
	if data != nil {
		rowsCount = int64(data.RowCount() + 1)
		colsCount = int64(data.ColCount())
	}
	if rowsCount < defaultMinRows {
		rowsCount = defaultMinRows
	}
	if colsCount < defaultMinCols {
		colsCount = defaultMinCols
	}

	targetRowCount := rowsCount + extraRowsBuffer
	targetColCount := colsCount + extraColsBuffer
	if tab != nil && tab.GridProperties != nil {
		if tab.GridProperties.RowCount > targetRowCount {
			targetRowCount = tab.GridProperties.RowCount
		}
		if tab.GridProperties.ColumnCount > targetColCount {
			targetColCount = tab.GridProperties.ColumnCount
		}
	}

	return rowsCount, colsCount, targetRowCount, targetColCount
}

//nolint:gocritic // returns rowCount, colCount, err
func (s *Service) resizeTabGridOnly(
	ctx context.Context,
	spreadsheetID string,
	tab *sheets.SheetProperties,
	rowsCount, colsCount, targetRowCount, targetColCount int64,
) (int64, int64, error) {
	if tab != nil && tab.GridProperties != nil &&
		tab.GridProperties.RowCount >= rowsCount &&
		tab.GridProperties.ColumnCount >= colsCount {
		return tab.GridProperties.RowCount, tab.GridProperties.ColumnCount, nil
	}

	sheetID := int64(0)
	title := ""
	if tab != nil {
		sheetID = tab.SheetId
		title = tab.Title
	}

	retryErr := gcp.WithRetry(ctx, func() error {
		_, e := s.sheets.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: []*sheets.Request{{
				UpdateSheetProperties: &sheets.UpdateSheetPropertiesRequest{
					Properties: &sheets.SheetProperties{
						SheetId: sheetID,
						GridProperties: &sheets.GridProperties{
							RowCount:    targetRowCount,
							ColumnCount: targetColCount,
						},
					},
					Fields: "gridProperties.rowCount,gridProperties.columnCount",
				},
			}},
		}).Context(ctx).Do()
		if e != nil {
			return fmt.Errorf("sheets api resize: %w", e)
		}

		return nil
	})
	if retryErr != nil {
		return 0, 0, fmt.Errorf("sheets: resize tab %q: %w", title, retryErr)
	}

	return targetRowCount, targetColCount, nil
}

func (s *Service) clearTab(ctx context.Context, spreadsheetID, tabTitle string) error {
	tabRange := EscapeSheetTitle(tabTitle)
	err := gcp.WithRetry(ctx, func() error {
		_, e := s.sheets.Spreadsheets.Values.Clear(spreadsheetID, tabRange, &sheets.ClearValuesRequest{}).
			Context(ctx).
			Do()
		if e != nil {
			return fmt.Errorf("sheets api clear: %w", e)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("sheets: clear tab %q: %w", tabTitle, err)
	}

	return nil
}

func (s *Service) writeTabBatches(ctx context.Context, spreadsheetID, tabTitle string, data *Table) (int, error) {
	if data == nil {
		return 0, nil
	}

	batches := prepareValueUpdates(tabTitle, data)
	for _, b := range batches {
		err := gcp.WithRetry(ctx, func() error {
			_, e := s.sheets.Spreadsheets.Values.Update(spreadsheetID, b.Range, &sheets.ValueRange{
				Values: b.Values,
			}).ValueInputOption(b.ValueInputOption).Context(ctx).Do()
			if e != nil {
				return fmt.Errorf("sheets api update values: %w", e)
			}

			return nil
		})
		if err != nil {
			return 0, fmt.Errorf("sheets: write range %s (opt=%s): %w", b.Range, b.ValueInputOption, err)
		}
	}

	return data.RowCount(), nil
}

func (s *Service) applyRichLinks(ctx context.Context, spreadsheetID string, sheetID int64, table *Table) {
	linkReqs := buildRichLinkRequests(sheetID, table)
	if len(linkReqs) == 0 {
		return
	}

	_ = gcp.WithRetry(ctx, func() error {
		_, e := s.sheets.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
			Requests: linkReqs,
		}).Context(ctx).Do()
		if e != nil {
			return fmt.Errorf("sheets api update rich links: %w", e)
		}

		return nil
	})
}

func (s *Service) applyTabFormatting(ctx context.Context, spreadsheetID string, sheetID int64, bandedIDs []int64, spec TabSpec) {
	totalDataRows := int64(1)
	colsCount := int64(0)
	if spec.Data != nil {
		totalDataRows = int64(spec.Data.RowCount() + 1)
		colsCount = int64(spec.Data.ColCount())
	}
	formatReqs := buildFormatRequests(sheetID, bandedIDs, spec, totalDataRows, colsCount)
	if len(formatReqs) > 0 {
		_ = gcp.WithRetry(ctx, func() error {
			_, e := s.sheets.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
				Requests: formatReqs,
			}).Context(ctx).Do()
			if e != nil {
				return fmt.Errorf("sheets api batch update format: %w", e)
			}

			return nil
		})
	}
}

func buildInitialSheets(tabs []TabSpec) []*sheets.Sheet {
	sheetDefs := make([]*sheets.Sheet, len(tabs))
	for i, t := range tabs {
		var frozenRows, frozenCols int64
		if !t.SkipFormatting {
			frozenRows = t.FrozenRows
			if frozenRows == 0 && t.Data != nil && len(t.Data.Headers) > 0 {
				frozenRows = 1
			}
			frozenCols = t.FrozenCols
		}
		sheetDefs[i] = &sheets.Sheet{
			Properties: &sheets.SheetProperties{
				Title: t.Title,
				GridProperties: &sheets.GridProperties{
					FrozenRowCount:    frozenRows,
					FrozenColumnCount: frozenCols,
				},
			},
		}
	}

	return sheetDefs
}

func (s *Service) populateCreatedTabs(ctx context.Context, created *sheets.Spreadsheet, tabs []TabSpec) ([]TabResult, error) {
	tabResults := make([]TabResult, len(tabs))
	for i, tabSpec := range tabs {
		var gid int64 = -1
		if i < len(created.Sheets) && created.Sheets[i].Properties != nil {
			gid = created.Sheets[i].Properties.SheetId
		}
		tabSpec.GID = gid

		res, err := s.ReplaceTab(ctx, created.SpreadsheetId, tabSpec)
		if err != nil {
			return nil, fmt.Errorf("sheets: populate tab %q: %w", tabSpec.Title, err)
		}
		tabResults[i] = *res
	}

	return tabResults, nil
}

func (s *Service) relocateToFolder(ctx context.Context, spreadsheetID, folderID string) {
	if folderID == "" || s.drive == nil {
		return
	}

	err := gcp.WithRetry(ctx, func() error {
		_, e := s.drive.Files.Update(spreadsheetID, nil).
			AddParents(folderID).
			RemoveParents("root").
			Context(ctx).
			Do()
		if e != nil {
			return fmt.Errorf("drive api update: %w", e)
		}

		return nil
	})
	if err != nil {
		s.log.WarnContext(ctx, "failed to move spreadsheet into target folder",
			slog.String("spreadsheet_id", spreadsheetID),
			slog.String("folder_id", folderID),
			slog.String("error", err.Error()),
		)
	}
}

func (s *Service) pruneStaleTabs(ctx context.Context, spreadsheetID string, currentSheets []*sheets.Sheet, desiredTabs []TabSpec) {
	keep := make(map[string]bool)
	for _, t := range desiredTabs {
		keep[t.Title] = true
	}

	var deleteReqs []*sheets.Request
	for _, sh := range currentSheets {
		if sh.Properties != nil && !keep[sh.Properties.Title] {
			deleteReqs = append(deleteReqs, &sheets.Request{
				DeleteSheet: &sheets.DeleteSheetRequest{
					SheetId: sh.Properties.SheetId,
				},
			})
		}
	}

	if len(deleteReqs) > 0 {
		_ = gcp.WithRetry(ctx, func() error {
			_, e := s.sheets.Spreadsheets.BatchUpdate(spreadsheetID, &sheets.BatchUpdateSpreadsheetRequest{
				Requests: deleteReqs,
			}).Context(ctx).Do()
			if e != nil {
				return fmt.Errorf("sheets api batch update delete: %w", e)
			}

			return nil
		})
	}
}
