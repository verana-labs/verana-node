import assert from "node:assert/strict";
import { createVeranaAminoTypes, createVeranaRegistry, veranaTypeUrls } from "../../src/signing";
import { MsgGrantOperatorAuthorization } from "../../src/codec/verana/de/v1/tx";
import { MsgStoreDigest } from "../../src/codec/verana/di/v1/tx";
import { MsgSelfCreateParticipant, MsgStartParticipantOP } from "../../src/codec/verana/pp/v1/tx";
import { ParticipantRole } from "../../src/codec/verana/pp/v1/types";
import {
  MsgCreateExchangeRate,
  MsgGrantExchangeRateAuthorization,
  MsgRevokeExchangeRateAuthorization,
  MsgSetExchangeRateState,
  MsgUpdateExchangeRate,
} from "../../src/codec/verana/xr/v1/tx";

const registry = createVeranaRegistry();
const amino = createVeranaAminoTypes() as any;

const requiredMappings = [
  "MsgCreateCorporation",
  "MsgCreateEcosystem",
  "MsgAddGovernanceFrameworkDocument",
  "MsgCreateCredentialSchema",
  "MsgSelfCreateParticipant",
  "MsgReclaimTrustDepositYield",
  "MsgGrantOperatorAuthorization",
  "MsgStoreDigest",
  "MsgCreateExchangeRate",
  "MsgUpdateExchangeRate",
  "MsgSetExchangeRateState",
  "MsgGrantExchangeRateAuthorization",
  "MsgRevokeExchangeRateAuthorization",
] as const;

for (const key of requiredMappings) {
  assert.ok(registry.lookupType(veranaTypeUrls[key]), `missing registry mapping for ${key}`);
  assert.ok(amino.register[veranaTypeUrls[key]], `missing amino mapping for ${key}`);
}

const deConverter = amino.register[veranaTypeUrls.MsgGrantOperatorAuthorization];
const deMsg = MsgGrantOperatorAuthorization.fromPartial({
  corporation: "verana1authority0000000000000000000000000000000",
  operator: "verana1operator0000000000000000000000000000000",
  grantee: "verana1grantee00000000000000000000000000000000",
  msgTypes: [veranaTypeUrls.MsgCreateEcosystem, veranaTypeUrls.MsgSelfCreateParticipant],
  expiration: new Date("2026-04-01T12:00:00.123Z"),
  authzSpendLimit: [{ denom: "uvna", amount: "42" }],
  authzSpendLimitPeriod: { seconds: 3600, nanos: 5 },
  withFeegrant: true,
  feegrantSpendLimit: [{ denom: "uvna", amount: "7" }],
  feegrantSpendLimitPeriod: { seconds: 7200, nanos: 0 },
});
const deRoundTrip = deConverter.fromAmino(deConverter.toAmino(deMsg));
assert.equal(deRoundTrip.msgTypes.length, 2);
assert.equal(deRoundTrip.authzSpendLimitPeriod?.seconds, 3600);
assert.equal(deRoundTrip.authzSpendLimitPeriod?.nanos, 5);
assert.equal(deRoundTrip.withFeegrant, true);
assert.equal(deRoundTrip.feegrantSpendLimitPeriod?.seconds, 7200);

// The chain's aminojson writes a Duration as a string of total nanoseconds
// (x/tx/signing/aminojson marshalDuration). The sign bytes break if this drifts
// back to a {seconds, nanos} object, so pin the wire shape, not just the
// round trip.
const deAmino = deConverter.toAmino(deMsg);
assert.equal(deAmino.authz_spend_limit_period, "3600000000005");
assert.equal(deAmino.feegrant_spend_limit_period, "7200000000000");

// A one-year period is past 2^53 nanoseconds, so the conversion has to keep
// 64-bit precision instead of going through a JS number.
const deLongPeriod = deConverter.toAmino(
  MsgGrantOperatorAuthorization.fromPartial({
    authzSpendLimitPeriod: { seconds: 31536000, nanos: 0 },
  }),
);
assert.equal(deLongPeriod.authz_spend_limit_period, "31536000000000000");

// A period that is set but zero is still written; only an absent one is omitted.
const deZeroPeriod = deConverter.toAmino(
  MsgGrantOperatorAuthorization.fromPartial({ authzSpendLimitPeriod: { seconds: 0, nanos: 0 } }),
);
assert.equal(deZeroPeriod.authz_spend_limit_period, "0");
assert.ok(
  !("authz_spend_limit_period" in deConverter.toAmino(MsgGrantOperatorAuthorization.fromPartial({}))),
  "an absent period must not be written",
);

const diConverter = amino.register[veranaTypeUrls.MsgStoreDigest];
const diMsg = MsgStoreDigest.fromPartial({
  corporation: "verana1corp0000000000000000000000000000000000",
  operator: "verana1operator0000000000000000000000000000000",
  digest: "sha256-abc123",
});
const diRoundTrip = diConverter.fromAmino(diConverter.toAmino(diMsg));
assert.equal(diRoundTrip.digest, "sha256-abc123");

const xrCreateConverter = amino.register[veranaTypeUrls.MsgCreateExchangeRate];
const xrCreateMsg = MsgCreateExchangeRate.fromPartial({
  authority: "verana1authority0000000000000000000000000000000",
  baseAssetType: 1,
  baseAsset: "EUR",
  quoteAssetType: 1,
  quoteAsset: "USD",
  rate: "1.0705",
  rateScale: 4,
  validityDuration: { seconds: 1800, nanos: 9 },
});
const xrCreateRoundTrip = xrCreateConverter.fromAmino(xrCreateConverter.toAmino(xrCreateMsg));
assert.equal(xrCreateRoundTrip.baseAsset, "EUR");
assert.equal(xrCreateRoundTrip.validityDuration?.seconds, 1800);
assert.equal(xrCreateRoundTrip.validityDuration?.nanos, 9);

const xrUpdateConverter = amino.register[veranaTypeUrls.MsgUpdateExchangeRate];
const xrUpdateMsg = MsgUpdateExchangeRate.fromPartial({
  operator: "verana1operator0000000000000000000000000000000",
  id: 12,
  rate: "1.0800",
});
const xrUpdateRoundTrip = xrUpdateConverter.fromAmino(xrUpdateConverter.toAmino(xrUpdateMsg));
assert.equal(xrUpdateRoundTrip.id, 12);
assert.equal(xrUpdateRoundTrip.rate, "1.0800");

const xrSetStateConverter = amino.register[veranaTypeUrls.MsgSetExchangeRateState];
const xrSetStateMsg = MsgSetExchangeRateState.fromPartial({
  authority: "verana1authority0000000000000000000000000000000",
  id: 12,
  state: false,
});
const xrSetStateRoundTrip = xrSetStateConverter.fromAmino(xrSetStateConverter.toAmino(xrSetStateMsg));
assert.equal(xrSetStateRoundTrip.id, 12);
assert.equal(xrSetStateRoundTrip.state, false);

const xrGrantConverter = amino.register[veranaTypeUrls.MsgGrantExchangeRateAuthorization];
const xrGrantMsg = MsgGrantExchangeRateAuthorization.fromPartial({
  authority: "verana1authority0000000000000000000000000000000",
  xrId: 12,
  operator: "verana1operator0000000000000000000000000000000",
  expiration: new Date("2026-09-01T08:30:00.456Z"),
  minInterval: { seconds: 600, nanos: 7 },
  maxDeviationBps: 250,
});
const xrGrantRoundTrip = xrGrantConverter.fromAmino(xrGrantConverter.toAmino(xrGrantMsg));
assert.equal(xrGrantRoundTrip.xrId, 12);
assert.equal(xrGrantRoundTrip.operator, "verana1operator0000000000000000000000000000000");
assert.equal(xrGrantRoundTrip.minInterval?.seconds, 600);
assert.equal(xrGrantRoundTrip.minInterval?.nanos, 7);
assert.equal(xrGrantRoundTrip.maxDeviationBps, 250);

const xrRevokeConverter = amino.register[veranaTypeUrls.MsgRevokeExchangeRateAuthorization];
const xrRevokeMsg = MsgRevokeExchangeRateAuthorization.fromPartial({
  authority: "verana1authority0000000000000000000000000000000",
  xrId: 12,
  operator: "verana1operator0000000000000000000000000000000",
});
const xrRevokeRoundTrip = xrRevokeConverter.fromAmino(xrRevokeConverter.toAmino(xrRevokeMsg));
assert.equal(xrRevokeRoundTrip.xrId, 12);
assert.equal(xrRevokeRoundTrip.operator, "verana1operator0000000000000000000000000000000");

const permConverter = amino.register[veranaTypeUrls.MsgSelfCreateParticipant];
const permMsg = MsgSelfCreateParticipant.fromPartial({
  corporation: "verana1authority0000000000000000000000000000000",
  operator: "verana1operator0000000000000000000000000000000",
  role: ParticipantRole.VERIFIER,
  validatorParticipantId: 17,
  did: "did:verana:test:perm",
  verificationFees: 0,
  validationFees: 0,
  vsOperator: "verana1vsoperator0000000000000000000000000000",
  vsOperatorAuthzMsgTypes: ["/verana.pp.v1.MsgCreateOrUpdateParticipantSession"],
  vsOperatorAuthzSpendLimit: [{ denom: "uvna", amount: "15" }],
  vsOperatorAuthzWithFeegrant: true,
  vsOperatorAuthzFeeSpendLimit: [{ denom: "uvna", amount: "5" }],
  vsOperatorAuthzPeriod: { seconds: 5400, nanos: 11 },
});
const permRoundTrip = permConverter.fromAmino(permConverter.toAmino(permMsg));
assert.equal(permRoundTrip.vsOperatorAuthzMsgTypes?.[0], "/verana.pp.v1.MsgCreateOrUpdateParticipantSession");
assert.equal(permRoundTrip.vsOperatorAuthzPeriod?.seconds, 5400);
assert.equal(permRoundTrip.vsOperatorAuthzPeriod?.nanos, 11);

const startVpConverter = amino.register[veranaTypeUrls.MsgStartParticipantOP];
const startVpMsg = MsgStartParticipantOP.fromPartial({
  corporation: "verana1authority0000000000000000000000000000000",
  operator: "verana1operator0000000000000000000000000000000",
  role: ParticipantRole.VALIDATOR,
  validatorParticipantId: 21,
  did: "did:verana:test:vp",
  validationFees: { value: 100 },
  issuanceFees: { value: 200 },
  verificationFees: { value: 300 },
  vsOperatorAuthzPeriod: { seconds: 900, nanos: 2 },
});
const startVpRoundTrip = startVpConverter.fromAmino(startVpConverter.toAmino(startVpMsg));
assert.equal(startVpRoundTrip.validationFees?.value, 100);
assert.equal(startVpRoundTrip.issuanceFees?.value, 200);
assert.equal(startVpRoundTrip.verificationFees?.value, 300);
assert.equal(startVpRoundTrip.vsOperatorAuthzPeriod?.seconds, 900);
assert.equal(startVpRoundTrip.vsOperatorAuthzPeriod?.nanos, 2);

console.log("signing surface check passed");
