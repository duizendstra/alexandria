package stackref

import "github.com/pulumi/pulumi/sdk/v3/go/pulumi"

// RequireString reads a string output from a stack reference.
// Returns the empty string when the key is absent or nil (e.g. the
// referenced stack is not deployed yet) — unlike the SDK's
// GetStringOutput, which fails the deployment on missing keys.
func RequireString(ref *pulumi.StackReference, key string) pulumi.StringOutput {
	out := ref.GetOutput(pulumi.String(key)).ApplyT(func(v any) string {
		s, _ := v.(string)

		return s
	})

	res, ok := out.(pulumi.StringOutput)
	if !ok {
		// Unreachable: ApplyT with a string-returning callback always
		// yields a StringOutput. Kept total for the type system.
		return pulumi.String("").ToStringOutput()
	}

	return res
}
