import Long from "long";
import type { Duration } from "../../codec/google/protobuf/duration";
import type { OptionalUInt32 } from "../../codec/verana/cs/v1/tx";

// clean drops proto3 default-valued keys so the amino doc matches the chain's
// aminojson, which omits empty fields lacking amino.dont_omitempty. null is
// preserved: it is the sentinel for a present-but-empty dont_omitempty field
// (e.g. an empty Coin array), which the chain encodes as null.
export const clean = <T extends Record<string, any>>(o: T): T => {
  Object.keys(o).forEach((k) => {
    const v = o[k];
    if (
      v === undefined ||
      v === "" ||
      v === false ||
      v === 0 ||
      (Array.isArray(v) && v.length === 0)
    ) {
      delete o[k];
    }
  });
  return o;
};

export const u64ToStr = (v?: Long | string | number | null) => {
  if (v == null) return undefined;
  const value = Long.fromValue(v);
  return value.isZero() ? undefined : value.toString();
};

export const u64ToStrIfNonZero = (v?: Long | string | number | null) => {
  if (v == null) return undefined;
  const value = Long.fromValue(v);
  return value.isZero() ? undefined : value.toString();
};

export const strToU64 = (s?: string | null) =>
  s != null ? Long.fromString(s) : undefined;

export const u32ToAmino = (n?: number | null) =>
  n == null ? undefined : (((n >>> 0) === 0) ? undefined : (n >>> 0));

export const pickOptionalUInt32 = (v: any): OptionalUInt32 | undefined => {
  if (v === undefined || v === null) return undefined;
  if (typeof v === "string" && v.trim() === "") return undefined;
  const n = Number(v);
  if (!Number.isFinite(n)) return undefined;
  return { value: n >>> 0 };
};

export const pickU32 = pickOptionalUInt32;

export const toOptU32Amino = (m?: { value: number } | undefined) => {
  if (!m) return undefined;
  const value = Number(m.value) >>> 0;
  return value === 0 ? {} : { value };
};

export const fromOptU32Amino = (x: any): OptionalUInt32 | undefined => {
  if (x == null) return undefined;
  if (typeof x === "object" && x.value == null) return { value: 0 };

  const n = typeof x === "object" ? x.value : x;
  if (n === undefined || n === null) return undefined;
  if (typeof n === "string" && n.trim() === "") return undefined;

  return { value: Number(n) >>> 0 };
};

export const dateToIsoAmino = (d?: Date | null) => {
  if (d == null) return undefined;
  return d
    .toISOString()
    .replace(/\.000Z$/, "Z")
    .replace(/(\.\d*?[1-9])0+Z$/, "$1Z");
};

export const isoToDate = (s?: string | null) =>
  s != null ? new Date(s) : undefined;

export const dateToAmino = dateToIsoAmino;
export const dateFromAmino = isoToDate;

// The chain (gogoproto.stdduration + aminojson) encodes a Duration as its total
// nanoseconds in a decimal string, e.g. 3600s -> "3600000000000". BigInt avoids
// the precision loss plain numbers hit for large durations.
export const durationToAmino = (d?: Duration | null) => {
  if (!d) return undefined;
  const seconds = BigInt(Long.fromValue(d.seconds).toString());
  const nanos = BigInt(d.nanos ?? 0);
  return (seconds * 1000000000n + nanos).toString();
};

export const aminoToDuration = (
  d?: string | number | { seconds?: string | number | null; nanos?: string | number | null } | null,
): Duration | undefined => {
  if (d == null) return undefined;
  // Canonical amino form: total-nanoseconds string. Object form kept for tolerance.
  if (typeof d === "string" || typeof d === "number") {
    const total = BigInt(d);
    const neg = total < 0n;
    const abs = neg ? -total : total;
    const seconds = abs / 1000000000n;
    const nanos = abs % 1000000000n;
    return { seconds: Number(neg ? -seconds : seconds), nanos: Number(neg ? -nanos : nanos) };
  }
  return {
    seconds: d.seconds == null ? 0 : Number(d.seconds),
    nanos: d.nanos == null ? 0 : Number(d.nanos),
  };
};
