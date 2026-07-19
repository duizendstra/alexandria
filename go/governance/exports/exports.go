package exports

// Stack output names — the governance API.
const (
	// OrgID is the GCP organization numeric ID.
	OrgID = "orgId"
	// BillingAccount is the billing account ID (e.g. 012345-6789AB-CDEF01).
	BillingAccount = "billingAccount"
	// RootFolderID is the numeric ID of the root folder.
	RootFolderID = "rootFolderId"
	// FolderIDs maps environment name → folder numeric ID.
	FolderIDs = "folderIds"
)

// TagKeyID returns the export name for a classification dimension's
// tag key ID. Dimensions are configuration, not code, so their export
// names are derived: dimension short name + "TagKeyId".
func TagKeyID(dimension string) string {
	return dimension + "TagKeyId"
}
