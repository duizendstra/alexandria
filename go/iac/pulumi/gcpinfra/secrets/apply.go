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

// ApplyContainers creates secrets and no versions.
//
// The secret is protected exactly as Apply protects one, and for the same
// reason: what a delete destroys is the version history and the IAM granted
// outside the stack, neither of which the stack can put back. That the stack
// never held the value makes the loss worse, not smaller — nothing here can
// recreate it.
//
// A container and a secret must not name the same secret. Pulumi would see two
// resources claiming one name, and whichever ran second would take the first
// one's version policy with it.
func ApplyContainers(
	ctx *pulumi.Context,
	projectID pulumi.StringOutput,
	containers []Container,
	deps []pulumi.Resource,
	opts ...lifecycle.Option,
) error {
	if err := ValidateContainers(containers); err != nil {
		return err
	}

	ephemeral := lifecycle.IsEphemeral(opts...)

	deletionPolicy := "PREVENT"
	if ephemeral {
		deletionPolicy = "DELETE"
	}

	for _, c := range containers {
		labels := make(pulumi.StringMap, len(c.Labels))
		for k, v := range c.Labels {
			labels[k] = pulumi.String(v)
		}

		_, err := secretmanager.NewSecret(ctx, c.Name, &secretmanager.SecretArgs{
			Project:  projectID,
			SecretId: pulumi.String(c.Name),
			Replication: &secretmanager.SecretReplicationArgs{
				Auto: &secretmanager.SecretReplicationAutoArgs{},
			},
			Labels:             labels,
			DeletionPolicy:     pulumi.String(deletionPolicy),
			DeletionProtection: pulumi.Bool(!ephemeral),
		}, pulumi.DependsOn(deps), lifecycle.Protect(opts...))
		if err != nil {
			return fmt.Errorf("create secret container %s: %w", c.Name, err)
		}
	}

	return nil
}
