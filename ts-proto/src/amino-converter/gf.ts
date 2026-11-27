import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgAddGovernanceFrameworkDocument,
  MsgIncreaseActiveGovernanceFrameworkVersion,
} from "../codec/verana/gf/v1/tx";
import { clean, u64ToStr } from "./util/helpers";

// MOD-GF Msgs extracted from x/tr in v4-rc2. ecosystem_id is the optional
// polymorphic subject discriminator: when zero/absent, the target is the
// signing corporation's own CGF; when set, the target is that Ecosystem.

export const MsgAddGovernanceFrameworkDocumentAminoConverter: AminoConverter = {
  aminoType: "verana/x/gf/MsgAddGovernanceFrameworkDocument",
  toAmino: ({
    corporation,
    signer,
    ecosystemId,
    docLanguage,
    docUrl,
    docDigestSri,
    version,
    network,
  }: MsgAddGovernanceFrameworkDocument) => clean({
    corporation: corporation || undefined,
    signer: signer || undefined,
    ecosystem_id: u64ToStr(ecosystemId as any),
    doc_language: docLanguage || undefined,
    doc_url: docUrl || undefined,
    doc_digest_sri: docDigestSri || undefined,
    version: version || undefined,
    network: network || undefined,
  }),
  fromAmino: (value: any) =>
    MsgAddGovernanceFrameworkDocument.fromPartial({
      corporation: value.corporation ?? "",
      signer: value.signer ?? "",
      ecosystemId: value.ecosystem_id != null ? Number(value.ecosystem_id) : 0,
      docLanguage: value.doc_language ?? "",
      docUrl: value.doc_url ?? "",
      docDigestSri: value.doc_digest_sri ?? "",
      version: value.version ?? 0,
      network: value.network ?? false,
    }),
};

export const MsgIncreaseActiveGovernanceFrameworkVersionAminoConverter: AminoConverter = {
  aminoType: "verana/x/gf/MsgIncreaseActiveGovernanceFrameworkVersion",
  toAmino: ({ corporation, signer, ecosystemId, network }: MsgIncreaseActiveGovernanceFrameworkVersion) => clean({
    corporation: corporation || undefined,
    signer: signer || undefined,
    ecosystem_id: u64ToStr(ecosystemId as any),
    network: network || undefined,
  }),
  fromAmino: (value: any) =>
    MsgIncreaseActiveGovernanceFrameworkVersion.fromPartial({
      corporation: value.corporation ?? "",
      signer: value.signer ?? "",
      ecosystemId: value.ecosystem_id != null ? Number(value.ecosystem_id) : 0,
      network: value.network ?? false,
    }),
};
