# Changelog

## [0.10.4](https://github.com/verana-labs/verana-node/compare/v0.10.3...v0.10.4) (2026-09-02)


### Bug Fixes

* **td,ec,pp,cs:** remove v3 leftover params and renumber param fields ([#53](https://github.com/verana-labs/verana-node/issues/53)) ([769b775](https://github.com/verana-labs/verana-node/commit/769b7756a1ed86e3a03dbc1df503b3cb91bfe5ec))

## [0.10.3](https://github.com/verana-labs/verana-node/compare/v0.10.2...v0.10.3) (2026-08-27)


### Bug Fixes

* **deps:** bump x/feegrant to v0.2.0 to fix panic on fee grants to fresh accounts ([#41](https://github.com/verana-labs/verana-node/issues/41)) ([9b0135b](https://github.com/verana-labs/verana-node/commit/9b0135bd62f6285a0eaec4c38e81f7afe819a88b))
* **pp:** align overlap checks of MSG-3, MSG-7, MSG-8 and MSG-14 ([#51](https://github.com/verana-labs/verana-node/issues/51)) ([5b5ed1e](https://github.com/verana-labs/verana-node/commit/5b5ed1e431dda7c27621d38fa981b6e4828ab717))
* **pp:** block new participants on unrepaid slashes and repay only the outstanding amount ([#50](https://github.com/verana-labs/verana-node/issues/50)) ([9e08400](https://github.com/verana-labs/verana-node/commit/9e08400832561ba85198ce743a72fc76776918b2))
* **pp:** make effective_from optional on root and self-create participant ([#48](https://github.com/verana-labs/verana-node/issues/48)) ([2e1ee14](https://github.com/verana-labs/verana-node/commit/2e1ee14d42858bc5b5b2e187885f334d42260a31))
* **pp:** rework start-OP overlap check to per-DID context ([#49](https://github.com/verana-labs/verana-node/issues/49)) ([60f1d01](https://github.com/verana-labs/verana-node/commit/60f1d0170fd22740a77dcc13cbc779b785f587c6))

## [0.10.2](https://github.com/verana-labs/verana-node/compare/v0.10.1...v0.10.2) (2026-08-13)


### Bug Fixes

* **cs,de,di,gf,pp,td,xr:** normalize REST routes to spec /&lt;mod&gt;/v1/* and regenerate OpenAPI ([#30](https://github.com/verana-labs/verana-node/issues/30)) ([228e711](https://github.com/verana-labs/verana-node/commit/228e7113a3147223da148aa67530d5169638cea6))
* **de:** treat unset VSOA expiration as never-expiring ([#37](https://github.com/verana-labs/verana-node/issues/37)) ([dca0d28](https://github.com/verana-labs/verana-node/commit/dca0d2888d160dd362fcb3f4ce0482a82a43c5e5))
* **pp:** drop signer field from trigger-resolver autocli flags ([#33](https://github.com/verana-labs/verana-node/issues/33)) ([0b52c89](https://github.com/verana-labs/verana-node/commit/0b52c89a3d09cf046c580018496c69c54ac0d9b6))
* **pp:** exclude revoked, slashed and repaid participants from start-OP overlap check ([#35](https://github.com/verana-labs/verana-node/issues/35)) ([b63e0b0](https://github.com/verana-labs/verana-node/commit/b63e0b0099830a453e69ef5ebbb5a7508153c7b3))
* **pp:** require effective_from on self-create participant ([#39](https://github.com/verana-labs/verana-node/issues/39)) ([0fd39dd](https://github.com/verana-labs/verana-node/commit/0fd39ddfddeb580242783963476c5cdec0a80943))

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
