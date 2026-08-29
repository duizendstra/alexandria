# Content-scan gate leak test (dummy PR, closed without merge)

This file exists only to prove that the content-scan CI gate (issue #317)
bites. It is **synthetic test data**: every string below is an obviously
fake, clearly-labelled placeholder. None of these values are, or ever were,
a real secret or a real identifier of any kind. This PR is closed without
merging once the failure is observed.

## Synthetic credential-shaped token (exercises `secret-scan`)

```
github_test_token = "ghp_FAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKEFAKE"
```

## Synthetic placeholder for the denylist half (exercises `denylist-scan`)

A throwaway maintainer-supplied pattern (e.g. matching the literal string
below) can be used, once the `CONTENT_DENYLIST_PATTERNS` secret exists, to
prove the denylist half also fails CI. This string is a generic fictional
placeholder (RFC 2606 reserved domain), not a real client, org, or product
identifier:

```
denylist-placeholder-string.example.invalid
```
