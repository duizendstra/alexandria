# Changelog

## [1.0.0](https://github.com/duizendstra/alexandria/compare/go/retry/v0.1.0...go/retry/v1.0.0) (2026-08-21)


### ⚠ BREAKING CHANGES

* **retry:** RoundTrip now returns the final retryable response together with a non-nil error wrapping the new ErrRetriesExceeded sentinel (match with errors.Is). The response body is left readable for direct RoundTrip callers; net/http.Client discards a response returned alongside an error, so through a client only the error surfaces. Exhausted transport-level errors now also wrap ErrRetriesExceeded in addition to the last error.

### Features

* add agentic diffract-review skill ([#15](https://github.com/duizendstra/alexandria/issues/15)) ([f31e870](https://github.com/duizendstra/alexandria/commit/f31e87060be0c66fc9fbff16f16817eeb436baaa))
* **contracts:** harvest schemas & seed 8-domain documentation vault ([#18](https://github.com/duizendstra/alexandria/issues/18)) ([7d9d358](https://github.com/duizendstra/alexandria/commit/7d9d3584fd08a8788954e8391f99553d83ea9f23))
* **go:** implement canonical monorepo hardening, SRE reliability, and performance optimizations ([#17](https://github.com/duizendstra/alexandria/issues/17)) ([7857d86](https://github.com/duizendstra/alexandria/commit/7857d86ab51d5e2ed70c3f0cad176d5b1b8a31c4))


### Bug Fixes

* **lint:** resolve golangci findings from stacked reliability PRs ([#62](https://github.com/duizendstra/alexandria/issues/62)) ([20ef2bc](https://github.com/duizendstra/alexandria/commit/20ef2bc2d1bff99479642c82e04627d854f107c7))
* **retry:** exhausted Transport retries now return an error; honor Retry-After ([#42](https://github.com/duizendstra/alexandria/issues/42)) ([075992b](https://github.com/duizendstra/alexandria/commit/075992b7d3a66b3d7b696e4e23b0f3a628dabcc4))
* **retry:** resolve safety, performance, and classification vulnerabilities ([#24](https://github.com/duizendstra/alexandria/issues/24)) ([fffbc81](https://github.com/duizendstra/alexandria/commit/fffbc81d44f9bd0b86b740d92e16c769fbef529a))
