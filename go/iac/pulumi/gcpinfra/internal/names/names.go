// Package names guards the configuration values that become Pulumi logical
// resource names.
//
// Pulumi requires every (type, name) URN in a stack to be unique. The
// building blocks in this module take the logical name straight from the
// caller's configuration, so a repeated value aborts the whole `pulumi up`
// with a duplicate-URN error — one the SDK test mocks never raise. Each
// block checks its own scope with Duplicate before it creates anything, so
// the repeat surfaces as a clear validation error at preview time instead.
package names

// Duplicate returns the first key that key yields twice for items, in slice
// order, and whether such a repeat exists.
func Duplicate[T any](items []T, key func(*T) string) (string, bool) {
	seen := make(map[string]struct{}, len(items))
	for i := range items {
		k := key(&items[i])
		if _, dup := seen[k]; dup {
			return k, true
		}

		seen[k] = struct{}{}
	}

	return "", false
}
