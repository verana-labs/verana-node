import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgGrantDelegateAuthorization,
  MsgRevokeDelegateAuthorization,
} from "../codec/verana/de/v1/tx";
import {
  aminoToDuration,
  clean,
  dateToIsoAmino,
  durationToAmino,
  isoToDate,
} from "./util/helpers";

export const MsgGrantDelegateAuthorizationAminoConverter: AminoConverter = {
  aminoType: "verana/x/de/MsgGrantDelegateAuthz",
  toAmino: (m: MsgGrantDelegateAuthorization) => clean({
    corporation: m.corporation || undefined,
    signer: m.signer || undefined,
    grantee: m.grantee || undefined,
    msg_types: m.msgTypes?.length ? m.msgTypes : undefined,
    expiration: dateToIsoAmino(m.expiration),
    authz_spend_limit: m.authzSpendLimit?.length ? m.authzSpendLimit : undefined,
    authz_spend_limit_period: durationToAmino(m.authzSpendLimitPeriod),
    with_feegrant: m.withFeegrant || undefined,
    feegrant_spend_limit: m.feegrantSpendLimit?.length ? m.feegrantSpendLimit : undefined,
    feegrant_spend_limit_period: durationToAmino(m.feegrantSpendLimitPeriod),
  }),
  fromAmino: (a: any): MsgGrantDelegateAuthorization =>
    MsgGrantDelegateAuthorization.fromPartial({
      corporation: a.corporation ?? "",
      signer: a.signer ?? "",
      grantee: a.grantee ?? "",
      msgTypes: a.msg_types ?? [],
      expiration: isoToDate(a.expiration),
      authzSpendLimit: a.authz_spend_limit ?? [],
      authzSpendLimitPeriod: aminoToDuration(a.authz_spend_limit_period),
      withFeegrant: a.with_feegrant ?? false,
      feegrantSpendLimit: a.feegrant_spend_limit ?? [],
      feegrantSpendLimitPeriod: aminoToDuration(a.feegrant_spend_limit_period),
    }),
};

export const MsgRevokeDelegateAuthorizationAminoConverter: AminoConverter = {
  aminoType: "verana/x/de/MsgRevokeDelegateAuthz",
  toAmino: ({ corporation, signer, grantee }: MsgRevokeDelegateAuthorization) => clean({
    corporation: corporation || undefined,
    signer: signer || undefined,
    grantee: grantee || undefined,
  }),
  fromAmino: (value: any) =>
    MsgRevokeDelegateAuthorization.fromPartial({
      corporation: value.corporation,
      signer: value.signer,
      grantee: value.grantee,
    }),
};
