package exports_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/governance/exports"
)

func ExampleOrgID() {
	fmt.Println(exports.OrgID)
	// Output:
	// orgId
}

func ExampleFolderIDs() {
	fmt.Println(exports.FolderIDs)
	// Output:
	// folderIds
}
