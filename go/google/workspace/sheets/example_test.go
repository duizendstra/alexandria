package sheets_test

import (
	"context"
	"fmt"
	"time"

	"github.com/duizendstra/alexandria/go/google/workspace/sheets"
)

// UserExportRecord represents a domain model exported from a web application.
type UserExportRecord struct {
	ID        int       `sheets:"User ID,width=80"`
	Name      string    `sheets:"Full Name,width=200"`
	Email     string    `sheets:"Email Address,width=220"`
	CreatedAt time.Time `sheets:"Registered"`
	Profile   string    `sheets:"Profile Link,formula"`
}

// ExampleService_CreateSpreadsheet demonstrates Use Case 1: Web application data export.
func ExampleService_CreateSpreadsheet() {
	ctx := context.Background()

	// In production, instantiate via client.NewSheetsService(ctx, sheets.Config{}, auth.WithAmbientADC()).
	var svc *sheets.Service

	records := []UserExportRecord{
		{
			ID:        1,
			Name:      "Alice Walker",
			Email:     "alice@example.com",
			CreatedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
			Profile:   `=HYPERLINK("https://app.example.com/users/1";"Open")`,
		},
	}

	table, err := sheets.FromStructs(records)
	if err != nil {
		fmt.Printf("table error: %v\n", err)

		return
	}

	spec := sheets.DocumentSpec{
		Title:    "User Directory Export",
		Locale:   "nl_NL",
		FolderID: "shared-drive-folder-id",
		Tabs: []sheets.TabSpec{
			{
				Title: "Users",
				Data:  table,
				Theme: sheets.ThemeCorporateNavy(),
			},
		},
	}

	if svc != nil {
		result, err := svc.CreateSpreadsheet(ctx, spec)
		if err != nil {
			fmt.Printf("export error: %v\n", err)

			return
		}
		fmt.Printf("Export ready at: %s\n", result.SpreadsheetURL)
	}

	// Output:
}

// ExampleService_ReplaceTab demonstrates Use Case 2: Idempotent migration tab projection.
func ExampleService_ReplaceTab() {
	ctx := context.Background()

	// In production, instantiate via client.NewSheetsService(ctx, sheets.Config{}, auth.WithImpersonation(...)).
	var svc *sheets.Service

	table := sheets.NewTable("Wave", "Rank", "Source Account", "Target Account", "Status", "Details Link")
	table.SetColumnWidth(2, 220)
	table.SetColumnWidth(3, 220)

	// Safe RAW literals for untrusted user inputs + explicit HYPERLINK formula.
	table.AddRow(
		sheets.Number(1),
		sheets.Number(1),
		sheets.Text("user.alpha@domain.com"),
		sheets.Text("user.alpha@target.com"),
		sheets.Text("READY"),
		sheets.Hyperlink("https://console.cloud.google.com/logs", "View Audit Log"),
	)

	spec := sheets.TabSpec{
		Title:           "wave_assignments",
		CreateIfMissing: true,
		FrozenRows:      1,
		Theme:           sheets.ThemeModernSlate(),
		Data:            table,
	}

	if svc != nil {
		result, err := svc.ReplaceTab(ctx, "existing-spreadsheet-id", spec)
		if err != nil {
			fmt.Printf("sync error: %v\n", err)

			return
		}
		fmt.Printf("Tab %s synced: %d rows\n", result.Title, result.RowsWritten)
	}

	// Output:
}
