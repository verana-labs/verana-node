# Changelog

## [0.10.1](https://github.com/verana-labs/verana-node/compare/v0.10.0...v0.10.1) (2026-08-06)


### Features

* add base CI/CD workflows ([#2](https://github.com/verana-labs/verana-node/issues/2)) ([78666f0](https://github.com/verana-labs/verana-node/commit/78666f0e2e3710d3ba2b3aa7f32437c538cc8e3f))
* **co,ec,pp:** enforce global DID ownership invariant ([#23](https://github.com/verana-labs/verana-node/issues/23)) ([f9142ea](https://github.com/verana-labs/verana-node/commit/f9142ea01cb1941c36c753762ce4bffb5b8c1224))
* **td:** set default trust_deposit_rate to 5% (0.05) ([#24](https://github.com/verana-labs/verana-node/issues/24)) ([053d03f](https://github.com/verana-labs/verana-node/commit/053d03f85e4e97f2501cb7d5b28724273ee16b47))


### Bug Fixes

* **cs:** align onboarding mode enum names with spec and fix create-schema CLI help ([#11](https://github.com/verana-labs/verana-node/issues/11)) ([9c54d31](https://github.com/verana-labs/verana-node/commit/9c54d31065ebfe545daafbc4378a9590006d9bd4))
* **cs:** remove SchemaAuthorizationPolicy (postponed to v5) ([#28](https://github.com/verana-labs/verana-node/issues/28)) ([9b2aa44](https://github.com/verana-labs/verana-node/commit/9b2aa44db2e1872f18b1a05a9a0557ba9537eeea))
* **de:** emit events when AUTHZ-CHECK paths debit or renew authorizations ([#13](https://github.com/verana-labs/verana-node/issues/13)) ([d3f2263](https://github.com/verana-labs/verana-node/commit/d3f226387c0331f2dbc0cb9f05503039eb66fa92))
* don't upload to S3 ([#4](https://github.com/verana-labs/verana-node/issues/4)) ([a11041b](https://github.com/verana-labs/verana-node/commit/a11041b1d90da58094d6749036d1f57dc430f348))
* economic modules hardening — delegation signer, participant genesis, and fund accounting (pp/de/td/xr/di) ([#7](https://github.com/verana-labs/verana-node/issues/7)) ([4c2bb1a](https://github.com/verana-labs/verana-node/commit/4c2bb1a374a51d1932153b74280df7e7f1fab8dc))
* **pp:** add amino.dont_omitempty so sign docs match amino wallet output ([#17](https://github.com/verana-labs/verana-node/issues/17)) ([cfa22be](https://github.com/verana-labs/verana-node/commit/cfa22be0408460a117adbee3983917bd0c6ff898))
* registry modules hardening - genesis, events, and SDK ergonomics (co/ec/gf/cs) ([#6](https://github.com/verana-labs/verana-node/issues/6)) ([670dfe9](https://github.com/verana-labs/verana-node/commit/670dfe91c42ca76be83310181b849864ebbc67e0))
* treat digest values as opaque strings ([#27](https://github.com/verana-labs/verana-node/issues/27)) ([eb3c2b1](https://github.com/verana-labs/verana-node/commit/eb3c2b175b14a8f7ea201a06d61a4f7dd0011675))
