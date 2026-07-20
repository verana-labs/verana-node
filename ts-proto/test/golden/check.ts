// Asserts every verana amino toAmino converter produces the exact canonical
// aminojson document the chain signs, so SIGN_MODE_LEGACY_AMINO_JSON verifies.
// The golden docs in amino-zero-golden.json are the output of the chain's own
// cosmossdk.io/x/tx/signing/aminojson encoder for a zero-valued message; the
// two populated cases below are likewise taken from that encoder. Run: npm run test:amino
import * as fs from "fs";
import * as path from "path";
import * as co from "../../src/amino-converter/co";
import * as cs from "../../src/amino-converter/cs";
import * as de from "../../src/amino-converter/de";
import * as di from "../../src/amino-converter/di";
import * as ec from "../../src/amino-converter/ec";
import * as gf from "../../src/amino-converter/gf";
import * as pp from "../../src/amino-converter/pp";
import * as td from "../../src/amino-converter/td";
import * as xr from "../../src/amino-converter/xr";

const mods: Record<string, any> = { co, cs, de, di, ec, gf, pp, td, xr };
const golden: Record<string, any> = JSON.parse(
  fs.readFileSync(path.join(__dirname, "amino-zero-golden.json"), "utf8"),
);

function canon(x: any): any {
  if (x === null || typeof x !== "object") return x;
  if (Array.isArray(x)) return x.map(canon);
  const o: any = {};
  for (const k of Object.keys(x).sort()) o[k] = canon(x[k]);
  return o;
}
const S = (x: any) => JSON.stringify(canon(x));

let fails = 0;
function expect(label: string, got: any, want: any) {
  if (S(got) === S(want)) return;
  fails++;
  console.log(`MISMATCH ${label}\n   got : ${JSON.stringify(canon(got))}\n   want: ${JSON.stringify(canon(want))}`);
}

// 1. Zero-value message: must match the chain's omitempty behaviour exactly.
let checked = 0;
for (const mod of Object.values(mods)) {
  for (const v of Object.values(mod) as any[]) {
    if (!v || typeof v !== "object" || !("aminoType" in v) || !("toAmino" in v)) continue;
    const want = golden[v.aminoType];
    if (want === undefined) { console.log("NO_GOLDEN", v.aminoType); fails++; continue; }
    checked++;
    expect(`${v.aminoType} (zero)`, v.toAmino({}), want);
  }
}

// 2. Populated messages: coins, uint64, enum, bool, timestamp, and Duration
//    (encoded by the chain as total-nanoseconds string).
expect("verana/x/pp/MsgCreateRootParticipant (populated)",
  pp.MsgCreateRootParticipantAminoConverter.toAmino({
    corporation: "verana1corp", operator: "verana1op", schemaId: 4, did: "did:web:x.example.com",
    effectiveFrom: new Date("2025-10-09T08:53:20Z"),
    validationFees: 100, issuanceFees: 200, verificationFees: 300, vsOperator: "verana1vsop",
    vsOperatorAuthzMsgTypes: ["/verana.pp.v1.MsgSetParticipantOPToValidated"],
    vsOperatorAuthzSpendLimit: [{ denom: "uvna", amount: "1000" }], vsOperatorAuthzWithFeegrant: true,
    vsOperatorAuthzFeeSpendLimit: [{ denom: "uvna", amount: "500" }],
    vsOperatorAuthzPeriod: { seconds: 3600, nanos: 0 },
  } as any),
  { corporation: "verana1corp", did: "did:web:x.example.com", effective_from: "2025-10-09T08:53:20Z",
    issuance_fees: "200", operator: "verana1op", schema_id: "4", validation_fees: "100", verification_fees: "300",
    vs_operator: "verana1vsop", vs_operator_authz_fee_spend_limit: [{ amount: "500", denom: "uvna" }],
    vs_operator_authz_msg_types: ["/verana.pp.v1.MsgSetParticipantOPToValidated"],
    vs_operator_authz_period: "3600000000000", vs_operator_authz_spend_limit: [{ amount: "1000", denom: "uvna" }],
    vs_operator_authz_with_feegrant: true });

expect("verana/x/de/MsgGrantOpAuthorization (populated)",
  de.MsgGrantOperatorAuthorizationAminoConverter.toAmino({
    corporation: "verana1corp", operator: "verana1op", grantee: "verana1grantee",
    msgTypes: ["/verana.pp.v1.MsgStartParticipantOP"], authzSpendLimit: [{ denom: "uvna", amount: "1000" }],
    authzSpendLimitPeriod: { seconds: 86400, nanos: 0 }, expiration: new Date("2026-01-02T03:04:05Z"),
    withFeegrant: true, feegrantSpendLimit: [{ denom: "uvna", amount: "50" }],
  } as any),
  { authz_spend_limit: [{ amount: "1000", denom: "uvna" }], authz_spend_limit_period: "86400000000000",
    corporation: "verana1corp", expiration: "2026-01-02T03:04:05Z", feegrant_spend_limit: [{ amount: "50", denom: "uvna" }],
    grantee: "verana1grantee", msg_types: ["/verana.pp.v1.MsgStartParticipantOP"], operator: "verana1op",
    with_feegrant: true });

console.log(`\nzero-checked=${checked} failures=${fails}`);
process.exit(fails ? 1 : 0);
