package secrets

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/lifecycle"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/secretmanager"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

// Apply creates Secret Manager secrets seeded with the given values.
// On every `pulumi up` the secret data is compared against the supplied
// value; if it changed a new version is created. The caller's secret
// source is the single source of truth — do not rotate secrets outside it.
//
// The secret is protected unless the caller passes lifecycle.Ephemeral. The
// caller can re-supply the value, but not the version history, the IAM granted
// on the secret outside the stack, or the reads that fail while a replacement
// is being created — and Secret Manager deletion has no undo.
//
// The version deliberately stays unprotected: its logical name is fixed, and
// rotation works by replacing it.
func Apply(ctx *pulumi.Context, projectID pulumi.StringOutput, ss []Secret, deps []pulumi.Resource, opts ...lifecycle.Option) error {
	if err := ValidateAll(ss); err != nil {
		return err
	}

	ephemeral := lifecycle.IsEphemeral(opts...)

	deletionPolicy := "PREVENT"
	if ephemeral {
		deletionPolicy = "DELETE"
	}

	for _, s := range ss {
		secret, err := secretmanager.NewSecret(ctx, s.Name, &secretmanager.SecretArgs{
			Project:  projectID,
			SecretId: pulumi.String(s.Name),
			Replication: &secretmanager.SecretReplicationArgs{
				Auto: &secretmanager.SecretReplicationAutoArgs{},
			},
			DeletionPolicy:     pulumi.String(deletionPolicy),
			DeletionProtection: pulumi.Bool(!ephemeral),
		}, pulumi.DependsOn(deps), lifecycle.Protect(opts...))
		if err != nil {
			return fmt.Errorf("create secret %s: %w", s.Name, err)
		}

		secretData, ok := pulumi.ToSecret(pulumi.String(s.Value)).(pulumi.StringOutput)
		if !ok {
			return fmt.Errorf("%w %q", ErrSecretDataType, s.Name)
		}

		_, err = secretmanager.NewSecretVersion(ctx, s.Name+"-v1", &secretmanager.SecretVersionArgs{
			Secret:     secret.ID(),
			SecretData: secretData,
		})
		if err != nil {
			return fmt.Errorf("create secret version %s: %w", s.Name, err)
		}
	}

	return nil
}
