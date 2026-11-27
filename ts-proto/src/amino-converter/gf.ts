import type { AminoConverter } from "@cosmjs/stargate";
import {
  MsgAcceptCorporationGovernanceFramework,
  MsgAcceptGovernanceFrameworkVersion,
  MsgAddGovernanceFrameworkDocument,
  MsgIncreaseActiveGovernanceFrameworkVersion,
} from "../codec/verana/gf/v1/tx";
import { aminoToDuration, clean, durationToAmino, u64ToStrIfNonZero } from "./util/helpers";

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
    ecosystem_id: u64ToStrIfNonZero(ecosystemId as any),
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
  toAmino: ({ corporation, signer, ecosystemId, network, acceptanceGrace }: MsgIncreaseActiveGovernanceFrameworkVersion) => clean({
    corporation: corporation || undefined,
    signer: signer || undefined,
    ecosystem_id: u64ToStrIfNonZero(ecosystemId as any),
    network: network || undefined,
    acceptance_grace: durationToAmino(acceptanceGrace),
  }),
  fromAmino: (value: any) =>
    MsgIncreaseActiveGovernanceFrameworkVersion.fromPartial({
      corporation: value.corporation ?? "",
      signer: value.signer ?? "",
      ecosystemId: value.ecosystem_id != null ? Number(value.ecosystem_id) : 0,
      network: value.network ?? false,
      acceptanceGrace: aminoToDuration(value.acceptance_grace),
    }),
};

export const MsgAcceptGovernanceFrameworkVersionAminoConverter: AminoConverter = {
  aminoType: "verana/x/gf/MsgAcceptGFVersion",
  toAmino: ({ corporation, signer, ecosystemId, network, version }: MsgAcceptGovernanceFrameworkVersion) => clean({
    corporation: corporation || undefined,
    signer: signer || undefined,
    ecosystem_id: u64ToStrIfNonZero(ecosystemId as any),
    network: network || undefined,
    version: version || undefined,
  }),
  fromAmino: (value: any) =>
    MsgAcceptGovernanceFrameworkVersion.fromPartial({
      corporation: value.corporation ?? "",
      signer: value.signer ?? "",
      ecosystemId: value.ecosystem_id != null ? Number(value.ecosystem_id) : 0,
      network: value.network ?? false,
      version: value.version ?? 0,
    }),
};

export const MsgAcceptCorporationGovernanceFrameworkAminoConverter: AminoConverter = {
  aminoType: "verana/x/gf/MsgAcceptCorporationGF",
  toAmino: ({ signer, corporationId, version }: MsgAcceptCorporationGovernanceFramework) => clean({
    signer: signer || undefined,
    corporation_id: u64ToStrIfNonZero(corporationId as any),
    version: version || undefined,
  }),
  fromAmino: (value: any) =>
    MsgAcceptCorporationGovernanceFramework.fromPartial({
      signer: value.signer ?? "",
      corporationId: value.corporation_id != null ? Number(value.corporation_id) : 0,
      version: value.version ?? 0,
    }),
};
