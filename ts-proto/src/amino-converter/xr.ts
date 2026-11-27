import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgCreateExchangeRate,
  MsgGrantExchangeRateAuthorization,
  MsgRevokeExchangeRateAuthorization,
  MsgSetExchangeRateState,
  MsgUpdateExchangeRate,
} from "../codec/verana/xr/v1/tx";
import {
  aminoToDuration,
  clean,
  dateToIsoAmino,
  durationToAmino,
  isoToDate,
  strToU64,
  u32ToAmino,
  u64ToStr,
} from "./util/helpers";

export const MsgCreateExchangeRateAminoConverter: AminoConverter = {
  aminoType: "verana/x/xr/MsgCreateExchangeRate",
  toAmino: (m: MsgCreateExchangeRate) => clean({
    signer: m.signer || undefined,
    base_asset_type: m.baseAssetType ?? 0,
    base_asset: m.baseAsset || undefined,
    quote_asset_type: m.quoteAssetType ?? 0,
    quote_asset: m.quoteAsset || undefined,
    rate: m.rate || undefined,
    rate_scale: u32ToAmino(m.rateScale),
    validity_duration: durationToAmino(m.validityDuration),
    state: m.state ? true : undefined,
  }),
  fromAmino: (a: any): MsgCreateExchangeRate =>
    MsgCreateExchangeRate.fromPartial({
      signer: a.signer ?? "",
      baseAssetType: a.base_asset_type ?? 0,
      baseAsset: a.base_asset ?? "",
      quoteAssetType: a.quote_asset_type ?? 0,
      quoteAsset: a.quote_asset ?? "",
      rate: a.rate ?? "",
      rateScale: a.rate_scale ?? 0,
      validityDuration: aminoToDuration(a.validity_duration),
      state: a.state ?? false,
    }),
};

export const MsgUpdateExchangeRateAminoConverter: AminoConverter = {
  aminoType: "verana/x/xr/MsgUpdateExchangeRate",
  toAmino: (m: MsgUpdateExchangeRate) => clean({
    signer: m.signer || undefined,
    id: u64ToStr(m.id),
    rate: m.rate || undefined,
  }),
  fromAmino: (a: any): MsgUpdateExchangeRate =>
    MsgUpdateExchangeRate.fromPartial({
      signer: a.signer ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      rate: a.rate ?? "",
    }),
};

export const MsgSetExchangeRateStateAminoConverter: AminoConverter = {
  aminoType: "verana/x/xr/MsgSetExchangeRateState",
  toAmino: (m: MsgSetExchangeRateState) => clean({
    signer: m.signer || undefined,
    id: u64ToStr(m.id),
    state: m.state ? true : undefined,
  }),
  fromAmino: (a: any): MsgSetExchangeRateState =>
    MsgSetExchangeRateState.fromPartial({
      signer: a.signer ?? "",
      id: strToU64(a.id) != null ? Number(strToU64(a.id)!.toString()) : 0,
      state: a.state ?? false,
    }),
};

export const MsgGrantExchangeRateAuthorizationAminoConverter: AminoConverter = {
  aminoType: "verana/x/xr/MsgGrantXrAuthz",
  toAmino: (m: MsgGrantExchangeRateAuthorization) => clean({
    signer: m.signer || undefined,
    xr_id: u64ToStr(m.xrId),
    feeder: m.feeder || undefined,
    expiration: dateToIsoAmino(m.expiration),
    min_interval: durationToAmino(m.minInterval),
    max_deviation_bps: u32ToAmino(m.maxDeviationBps),
  }),
  fromAmino: (a: any): MsgGrantExchangeRateAuthorization =>
    MsgGrantExchangeRateAuthorization.fromPartial({
      signer: a.signer ?? "",
      xrId: strToU64(a.xr_id) != null ? Number(strToU64(a.xr_id)!.toString()) : 0,
      feeder: a.feeder ?? "",
      expiration: isoToDate(a.expiration),
      minInterval: aminoToDuration(a.min_interval),
      maxDeviationBps: a.max_deviation_bps ?? 0,
    }),
};

export const MsgRevokeExchangeRateAuthorizationAminoConverter: AminoConverter = {
  aminoType: "verana/x/xr/MsgRevokeXrAuthz",
  toAmino: (m: MsgRevokeExchangeRateAuthorization) => clean({
    signer: m.signer || undefined,
    xr_id: u64ToStr(m.xrId),
    feeder: m.feeder || undefined,
  }),
  fromAmino: (a: any): MsgRevokeExchangeRateAuthorization =>
    MsgRevokeExchangeRateAuthorization.fromPartial({
      signer: a.signer ?? "",
      xrId: strToU64(a.xr_id) != null ? Number(strToU64(a.xr_id)!.toString()) : 0,
      feeder: a.feeder ?? "",
    }),
};
