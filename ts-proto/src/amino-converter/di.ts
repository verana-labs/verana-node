import type { AminoConverter } from "@cosmjs/stargate";
import { MsgStoreDigest } from "../codec/verana/di/v1/tx";
import { clean } from "./util/helpers";

export const MsgStoreDigestAminoConverter: AminoConverter = {
  aminoType: "verana/x/di/MsgStoreDigest",
  toAmino: (m: MsgStoreDigest) => clean({
    authority: m.authority || undefined,
    delegate: m.delegate || undefined,
    digest: m.digest || undefined,
  }),
  fromAmino: (a: any): MsgStoreDigest =>
    MsgStoreDigest.fromPartial({
      authority: a.authority ?? "",
      delegate: a.delegate ?? "",
      digest: a.digest ?? "",
    }),
};
