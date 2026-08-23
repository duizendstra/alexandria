package firestore

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/internal/names"
	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/lifecycle"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/firestore"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// DatabaseOutputs holds references to the created Firestore database.
type DatabaseOutputs struct {
	Name pulumi.StringOutput
}

// ApplyDatabase creates a Firestore Native database with Enterprise edition.
//
// The database is protected unless the caller passes lifecycle.Ephemeral.
// Without protection a rename does not lose the data — the provider's deletion
// policy defaults to ABANDON — but it drops the database out of the stack and
// the replacement then collides with it on the database ID.
func ApplyDatabase(
	ctx *pulumi.Context, projectID pulumi.StringOutput, cfg DatabaseConfig,
	deps []pulumi.Resource, opts ...lifecycle.Option,
) (*DatabaseOutputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ephemeral := lifecycle.IsEphemeral(opts...)

	// PREVENT rather than the provider's ABANDON default: ABANDON succeeds by
	// dropping the database out of the stack without deleting it, so the next
	// up quietly creates an empty replacement beside the live data. A stack
	// that keeps its data should fail instead; a disposable one has to destroy.
	deletionPolicy, deleteProtection := "PREVENT", DeleteProtectionEnabled
	if ephemeral {
		deletionPolicy, deleteProtection = "DELETE", DeleteProtectionDisabled
	}

	if cfg.DeleteProtection != "" {
		deleteProtection = cfg.DeleteProtection
	}

	db, err := firestore.NewDatabase(ctx, "firestore-"+cfg.Name, &firestore.DatabaseArgs{
		Project:                       projectID,
		Name:                          pulumi.String(cfg.Name),
		LocationId:                    pulumi.String(cfg.Region),
		Type:                          pulumi.String("FIRESTORE_NATIVE"),
		DatabaseEdition:               pulumi.String("ENTERPRISE"),
		DeleteProtectionState:         pulumi.String(deleteProtection),
		DeletionPolicy:                pulumi.String(deletionPolicy),
		PointInTimeRecoveryEnablement: pulumi.String("POINT_IN_TIME_RECOVERY_ENABLED"),
		FirestoreDataAccessMode:       pulumi.String("DATA_ACCESS_MODE_ENABLED"),
	}, pulumi.DependsOn(deps), lifecycle.Protect(opts...))
	if err != nil {
		return nil, fmt.Errorf("create firestore database %s: %w", cfg.Name, err)
	}

	return &DatabaseOutputs{Name: db.Name}, nil
}

// ApplyDocuments seeds Firestore documents. Field changes are ignored after
// initial creation — the application manages config at runtime, which is also
// why the documents are protected unless the caller passes lifecycle.Ephemeral:
// their live contents come from the application, not from Fields, so a rename
// deletes state that no config can put back.
//
// Every DocumentID in docs must be unique, even across collections: it is
// the Pulumi logical name, and a repeat is rejected with
// ErrDuplicateDocumentID before any document is created.
func ApplyDocuments(
	ctx *pulumi.Context, projectID pulumi.StringOutput, dbName string,
	docs []DocumentConfig, deps []pulumi.Resource, opts ...lifecycle.Option,
) error {
	for i := range docs {
		if err := docs[i].Validate(); err != nil {
			return err
		}
	}

	if id, dup := names.Duplicate(docs, func(d *DocumentConfig) string { return d.DocumentID }); dup {
		return fmt.Errorf("%w %q", ErrDuplicateDocumentID, id)
	}

	for _, doc := range docs {
		_, err := firestore.NewDocument(ctx, "doc-"+doc.DocumentID, &firestore.DocumentArgs{
			Project:    projectID,
			Database:   pulumi.String(dbName),
			Collection: pulumi.String(doc.Collection),
			DocumentId: pulumi.String(doc.DocumentID),
			Fields:     pulumi.String(doc.Fields),
		}, pulumi.DependsOn(deps),
			pulumi.IgnoreChanges([]string{"fields"}), lifecycle.Protect(opts...))
		if err != nil {
			return fmt.Errorf("seed document %s/%s: %w", doc.Collection, doc.DocumentID, err)
		}
	}

	return nil
}
