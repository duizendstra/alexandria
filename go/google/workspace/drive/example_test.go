package drive_test

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"

	"github.com/duizendstra/alexandria/go/google/auth"
	"github.com/duizendstra/alexandria/go/google/workspace/drive"
	google_drive "google.golang.org/api/drive/v3"
)

// ExampleNew demonstrates how to construct and initialize the Workspace Drive service
// using resolved authentication credentials (such as service account impersonation).
func ExampleNew() {
	ctx := context.Background()
	targetSA := "workspace-scanner@my-gcp-project.iam.gserviceaccount.com"

	// 1. Resolve authentication options using our SRE-grade auth package.
	clientOpts, err := auth.ResolveClient(
		ctx,
		[]string{google_drive.DriveMetadataReadonlyScope},
		auth.WithServiceAccountImpersonation(targetSA),
	)
	if err != nil {
		fmt.Printf("failed to resolve authentication options: %v\n", err)
		return
	}

	// 2. Instantiate our Workspace Drive service wrapper.
	srv, err := drive.New(ctx, drive.Config{Logger: slog.Default()}, clientOpts...)
	if err != nil {
		fmt.Printf("failed to create drive service: %v\n", err)
		return
	}

	_ = srv
}

// ExampleScanner_Scan demonstrates how to execute a high-performance flat-listing crawl
// across a shared drive with automatic O(1) streaming callback processing.
func ExampleScanner_Scan() {
	ctx := context.Background()

	// 1. Create our Drive service wrapper (simulated setup).
	srv, err := drive.New(ctx, drive.Config{Logger: slog.Default()})
	if err != nil {
		fmt.Printf("failed to create service: %v\n", err)
		return
	}

	// 2. Initialize our specialized high-performance scanner.
	scanner := drive.NewScanner(
		drive.WithPageSize(100),
		drive.WithMaxPages(50),
		drive.WithQuery("mimeType = 'application/pdf' and trashed = false"),
		drive.WithCorpora("allDrives"),
	)

	// 3. Execute the flat scan, streaming retrieved file metadata to our callback.
	err = scanner.Scan(ctx, srv.RawService(), func(f *google_drive.File) error {
		fmt.Printf("found file: %s (ID: %s, Mime: %s)\n", f.Name, f.Id, f.MimeType)

		return nil
	})
	if err != nil {
		fmt.Printf("scanning execution failed: %v\n", err)
		return
	}
}

// ExampleService_FetchFile demonstrates how to download raw binary file content
// or export a Google Workspace Document (e.g., Google Doc) as a PDF.
func ExampleService_FetchFile() {
	ctx := context.Background()
	fileID := "1abc123-file-id"

	// 1. Create our Drive service wrapper.
	srv, err := drive.New(ctx, drive.Config{Logger: slog.Default()})
	if err != nil {
		fmt.Printf("failed to create service: %v\n", err)
		return
	}

	// 2. Download and export file contents (auto-export Google Doc to PDF if prefix matches).
	content, err := srv.FetchFile(
		ctx,
		fileID,
		"application/vnd.google-apps.document", // Source is a Google Doc.
		"application/pdf",                      // Target destination is a PDF.
	)
	if err != nil {
		fmt.Printf("failed to fetch file: %v\n", err)
		return
	}

	fmt.Printf("successfully downloaded %d bytes of file content\n", len(content))
}

// ExampleService_Upload demonstrates how to upload a new file resource
// into a specific parent Google Drive folder.
func ExampleService_Upload() {
	ctx := context.Background()
	parentFolderID := "1xyz789-folder-id"

	// 1. Create our Drive service wrapper.
	srv, err := drive.New(ctx, drive.Config{Logger: slog.Default()})
	if err != nil {
		fmt.Printf("failed to create service: %v\n", err)
		return
	}

	// 2. Prepare mock reader content.
	body := bytes.NewReader([]byte("Hello from SRE Alexandria!"))

	// 3. Perform file upload with custom properties.
	file, err := srv.Upload(
		ctx,
		"report.txt",
		parentFolderID,
		map[string]string{"type": "audit-log"},
		body,
	)
	if err != nil {
		fmt.Printf("upload failed: %v\n", err)
		return
	}

	fmt.Printf("successfully uploaded file %s (ID: %s)\n", file.Name, file.ID)
}
