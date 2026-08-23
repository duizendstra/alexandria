# Changelog

## [1.1.0](https://github.com/duizendstra/alexandria/compare/go/iac/pulumi/gcpinfra/v1.0.1...go/iac/pulumi/gcpinfra/v1.1.0) (2026-08-23)


### Bug Fixes

* **gcpinfra:** protect data-bearing resources by default, with an Ephemeral opt-out ([#285](https://github.com/duizendstra/alexandria/issues/285)) ([#295](https://github.com/duizendstra/alexandria/issues/295)) ([dbd4643](https://github.com/duizendstra/alexandria/commit/dbd4643059843405cc234a83d130493a3c995e4e))


### Documentation

* **gcpinfra:** add runnable examples for the Ephemeral opt-out ([#296](https://github.com/duizendstra/alexandria/issues/296)) ([c79479d](https://github.com/duizendstra/alexandria/commit/c79479df79b18cf2a69dfbe94001d9dc69c7ad26))

## [1.0.1](https://github.com/duizendstra/alexandria/compare/go/iac/pulumi/gcpinfra/v1.0.0...go/iac/pulumi/gcpinfra/v1.0.1) (2026-08-22)


### Bug Fixes

* **gcpinfra:** reject config values that would repeat a Pulumi logical name ([#248](https://github.com/duizendstra/alexandria/issues/248)) ([#282](https://github.com/duizendstra/alexandria/issues/282)) ([a3b5fbd](https://github.com/duizendstra/alexandria/commit/a3b5fbd667acd531f1684a8a914af488bbe0e4d9))

## [1.0.0](https://github.com/duizendstra/alexandria/compare/go/iac/pulumi/gcpinfra/v0.7.0...go/iac/pulumi/gcpinfra/v1.0.0) (2026-08-21)


### ⚠ BREAKING CHANGES

* **gcpinfra:** Config.Host and ErrHostRequired are removed; Apply gains a host parameter — Apply(ctx, projectID, cfg, host, channelIDs, deps). Cheap now: the block is one release old (v0.6.0) with no consumers yet.

### Features

* **gcpinfra:** add projects, secrets, serviceaccounts, and iambindings ([#64](https://github.com/duizendstra/alexandria/issues/64)) ([ad62aef](https://github.com/duizendstra/alexandria/commit/ad62aefffb9779323bfbf865b241af3cd1b27616))
* **gcpinfra:** add six building blocks (budgets, datasets, logsinks, connections, registries, triggers) ([f11f16d](https://github.com/duizendstra/alexandria/commit/f11f16d5b9239245bea2f76635298c06fb86e18f))
* **gcpinfra:** add six building blocks for finops, observability, and delivery stacks ([35acb07](https://github.com/duizendstra/alexandria/commit/35acb07a43b3b41155c832d99c115c29320840c9))
* **gcpinfra:** add uptimechecks building block ([e7336f1](https://github.com/duizendstra/alexandria/commit/e7336f171d841fe40939c847e571eec6312076be))
* **gcpinfra:** add uptimechecks building block ([0d216ab](https://github.com/duizendstra/alexandria/commit/0d216abb31ffacfddd766a1be9e9d1742a86bcb1)), closes [#125](https://github.com/duizendstra/alexandria/issues/125)
* **gcpinfra:** multi-container (sidecar) cloudrun service ([6d7a0c6](https://github.com/duizendstra/alexandria/commit/6d7a0c6cd7f9d8976796d7e3dc5c6f7e2040c24d))
* **gcpinfra:** multi-container (sidecar) cloudrun service ([613c4eb](https://github.com/duizendstra/alexandria/commit/613c4ebff18747c176880316fb63afba7cdcbdf4))
* **gcpinfra:** uptimechecks host as runtime pulumi.StringInput ([8812b80](https://github.com/duizendstra/alexandria/commit/8812b80414547c18f8702acc30b87fde65c1fe81)), closes [#129](https://github.com/duizendstra/alexandria/issues/129)
* **iac/pulumi/gcpinfra:** add GCP folder and tag-key Pulumi building blocks ([14edb5b](https://github.com/duizendstra/alexandria/commit/14edb5b5aba709dfe48a7723c182596d95d57ba5))
* **iac/pulumi/gcpinfra:** add GCP folder and tag-key Pulumi building blocks ([b6a4c33](https://github.com/duizendstra/alexandria/commit/b6a4c333788c4eac06a649cd7b88ecc4c5677509))
* **iac:** ingestion/transform primitives — gcpinfra +5 packages, stackref, passstore ([5727d7f](https://github.com/duizendstra/alexandria/commit/5727d7f09f06140152da49a80851d1f245207396))
* **iac:** ingestion/transform primitives — gcpinfra packages, stackref, passstore ([6c37ee7](https://github.com/duizendstra/alexandria/commit/6c37ee73d23dba3d799c77c0fa88bf1ea54c5a04))


### Bug Fixes

* **gcpinfra:** folders.Apply stops enforcing tier policy it does not own ([29116b6](https://github.com/duizendstra/alexandria/commit/29116b61366c6009457c864c5c72779cefeaa4a3))
* **gcpinfra:** optional explicit CPU limit on cloudrun configs ([8f4d9bc](https://github.com/duizendstra/alexandria/commit/8f4d9bc564696e6a306e363524b638c046f617c0))
* **gcpinfra:** optional explicit CPU limit on cloudrun service/job configs ([31951a7](https://github.com/duizendstra/alexandria/commit/31951a764e0e299774377744e54e0fc8b426c070))
