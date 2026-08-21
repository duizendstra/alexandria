# Changelog

## [1.0.0](https://github.com/duizendstra/alexandria/compare/go/observability/audit/v0.0.3...go/observability/audit/v1.0.0) (2026-08-21)


### ⚠ BREAKING CHANGES

* **audit:** audit.Entry.Time changed from string to time.Time (v0.x). The JSONL wire format is unchanged.

### Features

* **contracts:** harvest schemas & seed 8-domain documentation vault ([#18](https://github.com/duizendstra/alexandria/issues/18)) ([7d9d358](https://github.com/duizendstra/alexandria/commit/7d9d3584fd08a8788954e8391f99553d83ea9f23))
* **go:** implement canonical monorepo hardening, SRE reliability, and performance optimizations ([#17](https://github.com/duizendstra/alexandria/issues/17)) ([7857d86](https://github.com/duizendstra/alexandria/commit/7857d86ab51d5e2ed70c3f0cad176d5b1b8a31c4))
* **observability/audit:** add contracts protobuf conversions ([#216](https://github.com/duizendstra/alexandria/issues/216)) ([ede39ef](https://github.com/duizendstra/alexandria/commit/ede39ef542e60dc6b808a064a98ebbe0c6194e1c))


### Bug Fixes

* **audit:** Entry.Time is time.Time with stable RFC3339 wire format ([#49](https://github.com/duizendstra/alexandria/issues/49)) ([08bbb02](https://github.com/duizendstra/alexandria/commit/08bbb02d9ae43e0403fc8508aed30b5eb0d091fb))
* **audit:** ReadScorecard skips malformed lines instead of hanging ([3bf0e27](https://github.com/duizendstra/alexandria/commit/3bf0e27fd9e721b251946e8323679873de3b3b83))
