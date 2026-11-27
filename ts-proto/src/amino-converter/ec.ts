import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgArchiveEcosystem,
  MsgCreateEcosystem,
  MsgUpdateEcosystem,
} from "../codec/verana/ec/v1/tx";
import { clean, u64ToStr } from "./util/helpers";

export const MsgCreateEcosystemAminoConverter: AminoConverter = {
  aminoType: "verana/x/ec/MsgCreateEcosystem",
  toAmino: ({ corporation, signer, did, language, docUrl, docDigestSri }: MsgCreateEcosystem) => clean({
    corporation: corporation || undefined,
    signer: signer || undefined,
    did: did || undefined,
    language: language || undefined,
    doc_url: docUrl || undefined,
    doc_digest_sri: docDigestSri || undefined,
  }),
  fromAmino: (value: any) =>
    MsgCreateEcosystem.fromPartial({
      corporation: value.corporation ?? "",
      signer: value.signer ?? "",
      did: value.did ?? "",
      language: value.language ?? "",
      docUrl: value.doc_url ?? "",
      docDigestSri: value.doc_digest_sri ?? "",
    }),
};

export const MsgUpdateEcosystemAminoConverter: AminoConverter = {
  aminoType: "verana/x/ec/MsgUpdateEcosystem",
  toAmino: ({ corporation, signer, id, did }: MsgUpdateEcosystem) => clean({
    corporation: corporation || undefined,
    signer: signer || undefined,
    id: u64ToStr(id as any),
    did: did || undefined,
  }),
  fromAmino: (value: any) =>
    MsgUpdateEcosystem.fromPartial({
      corporation: value.corporation ?? "",
      signer: value.signer ?? "",
      id: value.id != null ? Number(value.id) : 0,
      did: value.did ?? "",
    }),
};

export const MsgArchiveEcosystemAminoConverter: AminoConverter = {
  aminoType: "verana/x/ec/MsgArchiveEcosystem",
  toAmino: ({ corporation, signer, id, archive }: MsgArchiveEcosystem) => clean({
    corporation: corporation || undefined,
    signer: signer || undefined,
    id: u64ToStr(id as any),
    archive: archive,
  }),
  fromAmino: (value: any) =>
    MsgArchiveEcosystem.fromPartial({
      corporation: value.corporation ?? "",
      signer: value.signer ?? "",
      id: value.id != null ? Number(value.id) : 0,
      archive: value.archive ?? false,
    }),
};
