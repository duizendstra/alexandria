package firestore

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/internal/names"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/firestore"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// DatabaseOutputs holds references to the created Firestore database.
type DatabaseOutputs struct {
	Name pulumi.StringOutput
}

// ApplyDatabase creates a Firestore Native database with Enterprise edition.
func ApplyDatabase(ctx *pulumi.Context, projectID pulumi.StringOutput, cfg DatabaseConfig, deps []pulumi.Resource) (*DatabaseOutputs, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	db, err := firestore.NewDatabase(ctx, "firestore-"+cfg.Name, &firestore.DatabaseArgs{
		Project:                       projectID,
		Name:                          pulumi.String(cfg.Name),
		LocationId:                    pulumi.String(cfg.Region),
		Type:                          pulumi.String("FIRESTORE_NATIVE"),
		DatabaseEdition:               pulumi.String("ENTERPRISE"),
		DeleteProtectionState:         pulumi.String(cfg.DeleteProtection),
		PointInTimeRecoveryEnablement: pulumi.String("POINT_IN_TIME_RECOVERY_ENABLED"),
		FirestoreDataAccessMode:       pulumi.String("DATA_ACCESS_MODE_ENABLED"),
	}, pulumi.DependsOn(deps))
	if err != nil {
		return nil, fmt.Errorf("create firestore database %s: %w", cfg.Name, err)
	}

	return &DatabaseOutputs{Name: db.Name}, nil
}

// ApplyDocuments seeds Firestore documents. Field changes are ignored after
// initial creation — the application manages config at runtime.
//
// Every DocumentID in docs must be unique, even across collections: it is
// the Pulumi logical name, and a repeat is rejected with
// ErrDuplicateDocumentID before any document is created.
func ApplyDocuments(ctx *pulumi.Context, projectID pulumi.StringOutput, dbName string, docs []DocumentConfig, deps []pulumi.Resource) error {
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
			pulumi.IgnoreChanges([]string{"fields"}))
		if err != nil {
			return fmt.Errorf("seed document %s/%s: %w", doc.Collection, doc.DocumentID, err)
		}
	}

	return nil
}
