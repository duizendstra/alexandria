# Changelog

## [1.0.1](https://github.com/duizendstra/alexandria/compare/go/governance/v1.0.0...go/governance/v1.0.1) (2026-08-22)


### Bug Fixes

* **gate:** block on an unknown or empty policy instead of failing open ([#246](https://github.com/duizendstra/alexandria/issues/246)) ([#273](https://github.com/duizendstra/alexandria/issues/273)) ([9a4d454](https://github.com/duizendstra/alexandria/commit/9a4d4541518c531bb0d4c4a491d258fa2443bfdb))

## [1.0.0](https://github.com/duizendstra/alexandria/compare/go/governance/v0.4.0...go/governance/v1.0.0) (2026-08-22)


### ⚠ BREAKING CHANGES

* **governance:** plan.NewStarter and plan.NewStandard gain an orgID parameter (mirroring NewEnterprise). Without it they could never satisfy validateScope at Organization scope, making starter and standard tiers undeployable at org level. iac/governance derives the org ID from the GCP parent for org-scope plans and gains mocked end-to-end tests for all three tiers at both scopes. Pins staged to go/governance v0.2.0 and gcpinfra v0.1.1 — tag both after merge to make them resolvable.

### Features

* **governance,google:** add policy gate engine and google drive admin/membership primitives ([#225](https://github.com/duizendstra/alexandria/issues/225)) ([1ee8190](https://github.com/duizendstra/alexandria/commit/1ee8190b8d4ff453760f8fc6cf22765e493c79a3))
* **governance,platform:** invariant checks and build provenance stamping ([#193](https://github.com/duizendstra/alexandria/issues/193)) ([187c31c](https://github.com/duizendstra/alexandria/commit/187c31ca0eeedca0377319dabd359bc9ff21d4ea))
* **governance:** add cloud-agnostic governance domain module ([1f3ea25](https://github.com/duizendstra/alexandria/commit/1f3ea253ff78e3799baef7e9f96a7b9ff31598e4))
* **governance:** add cloud-agnostic governance domain module ([403ae6d](https://github.com/duizendstra/alexandria/commit/403ae6d12f1ee09bf56abdc19ae832bc659cdba9))
* **governance:** NewStarter/NewStandard take orgID; org-scope tiers deploy ([ebb89e5](https://github.com/duizendstra/alexandria/commit/ebb89e5be851c1d768500af07e3816942b72d9d7))
* **platform:** agentic Go platform evolution with native fuzzing and modernization ([#223](https://github.com/duizendstra/alexandria/issues/223)) ([ab4a0e3](https://github.com/duizendstra/alexandria/commit/ab4a0e3edc90abf43563d74553092e101fbad6b3))

## [0.4.0](https://github.com/duizendstra/alexandria/compare/go/governance/v0.3.0...go/governance/v0.4.0) (2026-08-21)


### Features

* **governance,google:** add policy gate engine and google drive admin/membership primitives ([#225](https://github.com/duizendstra/alexandria/issues/225)) ([1ee8190](https://github.com/duizendstra/alexandria/commit/1ee8190b8d4ff453760f8fc6cf22765e493c79a3))
* **platform:** agentic Go platform evolution with native fuzzing and modernization ([#223](https://github.com/duizendstra/alexandria/issues/223)) ([ab4a0e3](https://github.com/duizendstra/alexandria/commit/ab4a0e3edc90abf43563d74553092e101fbad6b3))

## [0.3.0](https://github.com/duizendstra/alexandria/compare/go/governance/v0.2.0...go/governance/v0.3.0) (2026-08-21)


### Features

* **governance,platform:** invariant checks and build provenance stamping ([#193](https://github.com/duizendstra/alexandria/issues/193)) ([187c31c](https://github.com/duizendstra/alexandria/commit/187c31ca0eeedca0377319dabd359bc9ff21d4ea))
