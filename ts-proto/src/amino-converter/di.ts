import type { AminoConverter } from "@cosmjs/stargate";
import { MsgStoreDigest } from "../codec/verana/di/v1/tx";
import { clean } from "./util/helpers";

export const MsgStoreDigestAminoConverter: AminoConverter = {
  aminoType: "verana/x/di/MsgStoreDigest",
  toAmino: (m: MsgStoreDigest) => clean({
    corporation: m.corporation || undefined,
    signer: m.signer || undefined,
    digest: m.digest || undefined,
  }),
  fromAmino: (a: any): MsgStoreDigest =>
    MsgStoreDigest.fromPartial({
      corporation: a.corporation ?? "",
      signer: a.signer ?? "",
      digest: a.digest ?? "",
    }),
};
