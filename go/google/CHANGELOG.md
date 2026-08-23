# Changelog

## [0.5.0](https://github.com/duizendstra/alexandria/compare/go/google/v0.4.2...go/google/v0.5.0) (2026-08-23)


### Features

* **auth,slog-gcp:** add ValidateAccessAs and native insertId support ([#215](https://github.com/duizendstra/alexandria/issues/215)) ([ec445b5](https://github.com/duizendstra/alexandria/commit/ec445b519ae5acc821bf34d726d12df81bb948c2))
* **contracts:** harvest schemas & seed 8-domain documentation vault ([#18](https://github.com/duizendstra/alexandria/issues/18)) ([7d9d358](https://github.com/duizendstra/alexandria/commit/7d9d3584fd08a8788954e8391f99553d83ea9f23))
* **go:** implement canonical monorepo hardening, SRE reliability, and performance optimizations ([#17](https://github.com/duizendstra/alexandria/issues/17)) ([7857d86](https://github.com/duizendstra/alexandria/commit/7857d86ab51d5e2ed70c3f0cad176d5b1b8a31c4))
* **google/workspace/drive:** add WithDriveID functional option ([02e4175](https://github.com/duizendstra/alexandria/commit/02e4175ecb7511e20bca4fb03430fa9237a5d10b))
* **google/workspace/drive:** add WithDriveID functional option to scanner ([72a999d](https://github.com/duizendstra/alexandria/commit/72a999d3a5923182b21e3e3a8fc60542eff43257))
* **google/workspace:** add SRE-grade Google Drive package with doc.go and examples ([8af0f2f](https://github.com/duizendstra/alexandria/commit/8af0f2f2e16c8a92723daaace30156a713841a27))
* **google/workspace:** introduce SRE Google Drive package ([a8aa760](https://github.com/duizendstra/alexandria/commit/a8aa760d020e49785d77c210a9cdb3cf274daa7d))
* **google:** add workspace sheets package with safe partitioning and themes ([#237](https://github.com/duizendstra/alexandria/issues/237)) ([e51d871](https://github.com/duizendstra/alexandria/commit/e51d87123ea5b691dc9c1dc7a33810268b8032d0))
* **google:** uniform retry via transport, single Drive constructor, honest ValidateAccess ([#48](https://github.com/duizendstra/alexandria/issues/48)) ([dfce995](https://github.com/duizendstra/alexandria/commit/dfce99554a1bd140a6d298559dc73ea59acbbf7a))
* **governance,google:** add policy gate engine and google drive admin/membership primitives ([#225](https://github.com/duizendstra/alexandria/issues/225)) ([ff772d1](https://github.com/duizendstra/alexandria/commit/ff772d19757aed1097dbd8256f4325f4c93f1aa8))
* **harvest:** upstream workflow, pulumi/runner, and gcp resource managers ([#224](https://github.com/duizendstra/alexandria/issues/224)) ([3c3d711](https://github.com/duizendstra/alexandria/commit/3c3d71117aec7a4d80697de87f8a6eb448e48a1d))


### Bug Fixes

* **google/workspace/drive:** escape backslashes in Drive query literals ([#255](https://github.com/duizendstra/alexandria/issues/255)) ([#280](https://github.com/duizendstra/alexandria/issues/280)) ([7a693cf](https://github.com/duizendstra/alexandria/commit/7a693cf0b441d3ac53e67f29e6f85b96f809a22a))
* **modules:** repair broken version pins so published modules resolve ([#36](https://github.com/duizendstra/alexandria/issues/36)) ([bed13b9](https://github.com/duizendstra/alexandria/commit/bed13b98e878c06ddfaee8c8d4b659a4f6dfcfdc))
* **sheets,observability,agents:** use generic placeholders in examples ([#302](https://github.com/duizendstra/alexandria/issues/302)) ([77ae4da](https://github.com/duizendstra/alexandria/commit/77ae4da21cec1319ceed7486bb3435290112d59d))
* **sheets:** keep GID-addressed tabs on prune and surface delete errors ([#243](https://github.com/duizendstra/alexandria/issues/243)) ([#278](https://github.com/duizendstra/alexandria/issues/278)) ([54a9acb](https://github.com/duizendstra/alexandria/commit/54a9acbc7b70baae57bd2e84f5e803ba034acaab))
* **sheets:** write a mixed column per cell so text cells never go USER_ENTERED ([#244](https://github.com/duizendstra/alexandria/issues/244)) ([#279](https://github.com/duizendstra/alexandria/issues/279)) ([0790c86](https://github.com/duizendstra/alexandria/commit/0790c86b3ee4d072731e3269dbfb4e83f49f4474))
* **workspace/sheets:** order formatting before rich links and return errors ([#242](https://github.com/duizendstra/alexandria/issues/242)) ([d1f6a63](https://github.com/duizendstra/alexandria/commit/d1f6a63db67cfb3af44fb465d9473c4ea4a96172))

## [0.4.2](https://github.com/duizendstra/alexandria/compare/go/google/v0.4.1...go/google/v0.4.2) (2026-08-22)


### Bug Fixes

* **google/workspace/drive:** escape backslashes in Drive query literals ([#255](https://github.com/duizendstra/alexandria/issues/255)) ([#280](https://github.com/duizendstra/alexandria/issues/280)) ([ccf81fd](https://github.com/duizendstra/alexandria/commit/ccf81fd6ef7701379af9119b0a691d0c7a1b572c))
* **sheets:** keep GID-addressed tabs on prune and surface delete errors ([#243](https://github.com/duizendstra/alexandria/issues/243)) ([#278](https://github.com/duizendstra/alexandria/issues/278)) ([c40baa2](https://github.com/duizendstra/alexandria/commit/c40baa260f145eeb91d9900b5d341edceb4de7f0))
* **sheets:** write a mixed column per cell so text cells never go USER_ENTERED ([#244](https://github.com/duizendstra/alexandria/issues/244)) ([#279](https://github.com/duizendstra/alexandria/issues/279)) ([c3ce465](https://github.com/duizendstra/alexandria/commit/c3ce465ab8648386da0d9a0c06cdec74ee5a2efc))

## [0.4.1](https://github.com/duizendstra/alexandria/compare/go/google/v0.4.0...go/google/v0.4.1) (2026-08-22)


### Bug Fixes

* **workspace/sheets:** order formatting before rich links and return errors ([#242](https://github.com/duizendstra/alexandria/issues/242)) ([0482cec](https://github.com/duizendstra/alexandria/commit/0482cecf1d53bf74d7dfc825e3cc216bbcbdfb3d))

## [0.3.0](https://github.com/duizendstra/alexandria/compare/go/google/v0.2.0...go/google/v0.3.0) (2026-08-22)


### Features

* **auth,slog-gcp:** add ValidateAccessAs and native insertId support ([#215](https://github.com/duizendstra/alexandria/issues/215)) ([419bb9d](https://github.com/duizendstra/alexandria/commit/419bb9d557dae505eaee0b90f57c5fde7dd258ba))
* **contracts:** harvest schemas & seed 8-domain documentation vault ([#18](https://github.com/duizendstra/alexandria/issues/18)) ([7d9d358](https://github.com/duizendstra/alexandria/commit/7d9d3584fd08a8788954e8391f99553d83ea9f23))
* **go:** implement canonical monorepo hardening, SRE reliability, and performance optimizations ([#17](https://github.com/duizendstra/alexandria/issues/17)) ([7857d86](https://github.com/duizendstra/alexandria/commit/7857d86ab51d5e2ed70c3f0cad176d5b1b8a31c4))
* **google/workspace/drive:** add WithDriveID functional option ([02e4175](https://github.com/duizendstra/alexandria/commit/02e4175ecb7511e20bca4fb03430fa9237a5d10b))
* **google/workspace/drive:** add WithDriveID functional option to scanner ([72a999d](https://github.com/duizendstra/alexandria/commit/72a999d3a5923182b21e3e3a8fc60542eff43257))
* **google/workspace:** add SRE-grade Google Drive package with doc.go and examples ([8af0f2f](https://github.com/duizendstra/alexandria/commit/8af0f2f2e16c8a92723daaace30156a713841a27))
* **google/workspace:** introduce SRE Google Drive package ([a8aa760](https://github.com/duizendstra/alexandria/commit/a8aa760d020e49785d77c210a9cdb3cf274daa7d))
* **google:** uniform retry via transport, single Drive constructor, honest ValidateAccess ([#48](https://github.com/duizendstra/alexandria/issues/48)) ([e3c4a32](https://github.com/duizendstra/alexandria/commit/e3c4a32ee3824d632b66746fe28be79bbf84a366))
* **governance,google:** add policy gate engine and google drive admin/membership primitives ([#225](https://github.com/duizendstra/alexandria/issues/225)) ([1ee8190](https://github.com/duizendstra/alexandria/commit/1ee8190b8d4ff453760f8fc6cf22765e493c79a3))
* **harvest:** upstream workflow, pulumi/runner, and gcp resource managers ([#224](https://github.com/duizendstra/alexandria/issues/224)) ([401ad59](https://github.com/duizendstra/alexandria/commit/401ad599d713df00909102eb7c9852eccadc92cc))


### Bug Fixes

* **modules:** repair broken version pins so published modules resolve ([#36](https://github.com/duizendstra/alexandria/issues/36)) ([61deda7](https://github.com/duizendstra/alexandria/commit/61deda7e8686038218f3416fb979f773d3e5417b))

## [0.2.0](https://github.com/duizendstra/alexandria/compare/go/google/v0.1.0...go/google/v0.2.0) (2026-08-21)


### Features

* **governance,google:** add policy gate engine and google drive admin/membership primitives ([#225](https://github.com/duizendstra/alexandria/issues/225)) ([1ee8190](https://github.com/duizendstra/alexandria/commit/1ee8190b8d4ff453760f8fc6cf22765e493c79a3))
* **harvest:** upstream workflow, pulumi/runner, and gcp resource managers ([#224](https://github.com/duizendstra/alexandria/issues/224)) ([401ad59](https://github.com/duizendstra/alexandria/commit/401ad599d713df00909102eb7c9852eccadc92cc))

## [0.1.0](https://github.com/duizendstra/alexandria/compare/go/google/v0.0.3...go/google/v0.1.0) (2026-08-21)


### Features

* **auth,slog-gcp:** add ValidateAccessAs and native insertId support ([#215](https://github.com/duizendstra/alexandria/issues/215)) ([419bb9d](https://github.com/duizendstra/alexandria/commit/419bb9d557dae505eaee0b90f57c5fde7dd258ba))
* **contracts:** harvest schemas & seed 8-domain documentation vault ([#18](https://github.com/duizendstra/alexandria/issues/18)) ([7d9d358](https://github.com/duizendstra/alexandria/commit/7d9d3584fd08a8788954e8391f99553d83ea9f23))
* **go:** implement canonical monorepo hardening, SRE reliability, and performance optimizations ([#17](https://github.com/duizendstra/alexandria/issues/17)) ([7857d86](https://github.com/duizendstra/alexandria/commit/7857d86ab51d5e2ed70c3f0cad176d5b1b8a31c4))
* **google/workspace/drive:** add WithDriveID functional option ([02e4175](https://github.com/duizendstra/alexandria/commit/02e4175ecb7511e20bca4fb03430fa9237a5d10b))
* **google/workspace/drive:** add WithDriveID functional option to scanner ([72a999d](https://github.com/duizendstra/alexandria/commit/72a999d3a5923182b21e3e3a8fc60542eff43257))
* **google/workspace:** add SRE-grade Google Drive package with doc.go and examples ([8af0f2f](https://github.com/duizendstra/alexandria/commit/8af0f2f2e16c8a92723daaace30156a713841a27))
* **google/workspace:** introduce SRE Google Drive package ([a8aa760](https://github.com/duizendstra/alexandria/commit/a8aa760d020e49785d77c210a9cdb3cf274daa7d))
* **google:** uniform retry via transport, single Drive constructor, honest ValidateAccess ([#48](https://github.com/duizendstra/alexandria/issues/48)) ([e3c4a32](https://github.com/duizendstra/alexandria/commit/e3c4a32ee3824d632b66746fe28be79bbf84a366))


### Bug Fixes

* **modules:** repair broken version pins so published modules resolve ([#36](https://github.com/duizendstra/alexandria/issues/36)) ([61deda7](https://github.com/duizendstra/alexandria/commit/61deda7e8686038218f3416fb979f773d3e5417b))
