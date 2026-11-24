/**
 * Journey: DE Grant PERM Delegate Authorization
 *
 * Grants delegate authorization from the authority account to a perm-specific
 * delegate for all PERM message types, plus TR and CS message types needed
 * for creating prerequisites in perm journeys.
 *
 * Uses: authority=index 10 (same as TR/CS), delegate=index 15
 *
 * Usage:
 *   npm run test:de-grant-perm-auth
 */

import {
  createWallet,
  createAccountFromMnemonic,
  createSigningClient,
  createQueryClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  fundAccount,
  config,
  acceptCorporationGovernanceFramework,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgGrantDelegateAuthorization } from "../../../src/codec/verana/de/v1/tx";
import { getActiveCorporation, getEcAuthzSetup, savePermAuthzSetup } from "../helpers/journeyResults";
import { MsgSubmitProposal, MsgVote, Exec } from "cosmjs-types/cosmos/group/v1/tx";
import { VoteOption } from "cosmjs-types/cosmos/group/v1/types";
import { MsgType } from "../../../src/codec/verana/de/v1/msg_type";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const DELEGATE_INDEX = 15;
// Corp group member (proposer/voter) — matches co-create's SIGNER_INDEX.
const SIGNER_INDEX = 10;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: DE Grant PERM Delegate Authorization");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load authority (Corporation policy_address) from EC authz setup
  console.log("Step 1: Loading authority from EC authz setup...");
  const ecSetup = getEcAuthzSetup();
  if (!ecSetup) {
    console.log("  No EC authz setup found. Run test:de-grant-auth first.");
    process.exit(1);
  }
  const authorityAddress = ecSetup.authorityAddress;
  console.log(`  Authority (corporation policy_address): ${authorityAddress}`);
  console.log();

  // Step 2: Create perm delegate wallet
  console.log("Step 2: Creating PERM delegate wallet...");
  const delegateWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, DELEGATE_INDEX);
  const cooluserWallet = await createWallet(COOLUSER_MNEMONIC);
  const delegateAccount = await getAccountInfo(delegateWallet);
  const cooluserAccount = await getAccountInfo(cooluserWallet);
  console.log(`  PERM Delegate: ${delegateAccount.address} (index ${DELEGATE_INDEX})`);
  console.log();

  // Step 3: Fund delegate
  console.log("Step 3: Funding PERM delegate...");
  const fundAmount = "5000000000uvna"; // 5000 VNA (enough for perm operations with trust deposits + fees)

  const fundResult = await fundAccount(
    COOLUSER_MNEMONIC,
    cooluserAccount.address,
    delegateAccount.address,
    fundAmount,
  );
  if (fundResult.code !== 0) {
    console.log(`  Failed to fund delegate: ${fundResult.rawLog}`);
    process.exit(1);
  }
  console.log(`  Funded delegate with ${fundAmount}`);

  // Wait for funding to confirm
  const queryClient = await createQueryClient();
  console.log("  Waiting for funding tx to confirm...");
  for (let i = 0; i < 30; i++) {
    try {
      const tx = await queryClient.getTx(fundResult.transactionHash);
      if (tx) {
        console.log(`  Funding confirmed at block ${tx.height}`);
        break;
      }
    } catch {}
    await new Promise((r) => setTimeout(r, 1000));
  }

  // Verify balance
  for (let i = 0; i < 20; i++) {
    const balance = await queryClient.getBalance(delegateAccount.address, config.denom);
    if (BigInt(balance.amount) > 0) {
      console.log(`  Delegate balance: ${balance.amount}${balance.denom}`);
      break;
    }
    await new Promise((r) => setTimeout(r, 1000));
  }
  queryClient.disconnect();
  console.log();

  // Step 4: Grant delegate authorization for all PERM + TR + CS message types
  console.log("Step 4: Granting delegate authorization...");

  const allMsgTypes = [
    // EC messages (for creating prerequisite Ecosystems)
    MsgType.MSG_TYPE_CREATE_ECOSYSTEM,
    MsgType.MSG_TYPE_UPDATE_ECOSYSTEM,
    MsgType.MSG_TYPE_ARCHIVE_ECOSYSTEM,
    // GF messages (moved from verana.tr.v1 to verana.gf.v1 in v4-rc2)
    MsgType.MSG_TYPE_ADD_GOVERNANCE_FRAMEWORK_DOCUMENT,
    MsgType.MSG_TYPE_INCREASE_ACTIVE_GOVERNANCE_FRAMEWORK_VERSION,
    // CS messages (for creating prerequisite CSs)
    MsgType.MSG_TYPE_CREATE_CREDENTIAL_SCHEMA,
    MsgType.MSG_TYPE_UPDATE_CREDENTIAL_SCHEMA,
    MsgType.MSG_TYPE_ARCHIVE_CREDENTIAL_SCHEMA,
    // PERM messages
    MsgType.MSG_TYPE_CREATE_ROOT_ACCREDITATION,
    MsgType.MSG_TYPE_SELF_CREATE_ACCREDITATION,
    MsgType.MSG_TYPE_SET_ACCREDITATION_EFFECTIVE_UNTIL,
    MsgType.MSG_TYPE_REVOKE_ACCREDITATION,
    MsgType.MSG_TYPE_START_ONBOARDING,
    MsgType.MSG_TYPE_RENEW_ONBOARDING,
    MsgType.MSG_TYPE_VALIDATE_ONBOARDING,
    MsgType.MSG_TYPE_WITHDRAW_ONBOARDING_REQUEST,
    // Note: MsgCreateOrUpdateParticipantSession is NOT DE-delegable
    // It uses VS operator authorization (AUTHZ-CHECK-3) instead
    MsgType.MSG_TYPE_SLASH_PARTICIPATION_TRUST_DEPOSIT,
    MsgType.MSG_TYPE_REPAY_SLASH,
    // [MOD-PP-MSG-15] Trigger Resolver
    MsgType.MSG_TYPE_TRIGGER_RESOLVER,
  ];

  // The corporation policy_address cannot sign directly; the corp's sole group
  // member (SIGNER_INDEX, matching co-create) submits an x/group proposal carrying
  // the inner MsgGrantDelegateAuthorization (delegate == corporation policy_address
  // = corporation acts alone) and votes YES with EXEC_TRY so the policy executes it.
  const signerWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, SIGNER_INDEX);
  const signerAccount = await getAccountInfo(signerWallet);
  const client = await createSigningClient(signerWallet);

  const innerMsgValue = MsgGrantDelegateAuthorization.encode(
    MsgGrantDelegateAuthorization.fromPartial({
      corporation: authorityAddress,
      delegate: authorityAddress,
      grantee: delegateAccount.address,
      msgTypes: allMsgTypes,
      withFeegrant: false,
    }),
  ).finish();

  const proposalMsg = {
    typeUrl: "/cosmos.group.v1.MsgSubmitProposal",
    value: MsgSubmitProposal.fromPartial({
      groupPolicyAddress: authorityAddress,
      proposers: [signerAccount.address],
      metadata: "Grant PERM delegate authz",
      title: "Grant PERM delegate authz",
      summary: "Grant delegate authorization for PERM/EC/GF/CS message types",
      messages: [{ typeUrl: typeUrls.MsgGrantDelegateAuthorization, value: innerMsgValue }],
      exec: Exec.EXEC_UNSPECIFIED,
    }),
  };

  try {
    // 4a: submit proposal
    const proposalFee = await calculateFeeWithSimulation(client, signerAccount.address, [proposalMsg], "Submit PERM grant proposal");
    const proposalResult = await signAndBroadcastWithRetry(client, signerAccount.address, [proposalMsg], proposalFee, "Submit PERM grant proposal");
    if (proposalResult.code !== 0) {
      if (String(proposalResult.rawLog).includes("already exists")) {
        savePermAuthzSetup(authorityAddress, delegateAccount.address);
        console.log("  Authorization already exists; saved setup.");
        client.disconnect();
        console.log("=".repeat(60));
        return;
      }
      console.log(`FAILED proposal: ${proposalResult.rawLog}`);
      process.exit(1);
    }
    let proposalId: bigint | undefined;
    for (const event of (proposalResult.events || [])) {
      for (const attr of event.attributes) {
        if (attr.key === "proposal_id") { proposalId = BigInt(String(attr.value).replace(/"/g, "")); break; }
      }
      if (proposalId !== undefined) break;
    }
    if (proposalId === undefined) { console.log("Could not extract proposal_id"); process.exit(1); }
    console.log(`  Proposal ID: ${proposalId}`);
    const qc2 = await createQueryClient();
    for (let i = 0; i < 30; i++) { try { if (await qc2.getTx(proposalResult.transactionHash)) break; } catch {} await new Promise((r) => setTimeout(r, 1000)); }
    qc2.disconnect();

    // 4b: vote YES with EXEC_TRY (auto-executes)
    const voteMsg = {
      typeUrl: "/cosmos.group.v1.MsgVote",
      value: MsgVote.fromPartial({ proposalId, voter: signerAccount.address, option: VoteOption.VOTE_OPTION_YES, exec: Exec.EXEC_TRY, metadata: "" }),
    };
    const voteFee = await calculateFeeWithSimulation(client, signerAccount.address, [voteMsg], "Vote YES PERM grant");
    const voteResult = await signAndBroadcastWithRetry(client, signerAccount.address, [voteMsg], voteFee, "Vote YES PERM grant");
    if (voteResult.code !== 0) { console.log(`FAILED vote: ${voteResult.rawLog}`); process.exit(1); }
    console.log(`  Voted YES + executed at block ${voteResult.height}; authorized ${allMsgTypes.length} msg types`);
    const qc3 = await createQueryClient();
    for (let i = 0; i < 30; i++) { try { if (await qc3.getTx(voteResult.transactionHash)) break; } catch {} await new Promise((r) => setTimeout(r, 1000)); }
    qc3.disconnect();

    const corp = getActiveCorporation();
    if (!corp) { console.log("No active corporation found. Run test:co-create first."); process.exit(1); }
    await acceptCorporationGovernanceFramework(delegateWallet, corp.corporationId);
    console.log("  Delegate accepted the corporation's governance framework");

    savePermAuthzSetup(authorityAddress, delegateAccount.address);
    console.log("  Saved perm-authz-setup");
  } catch (error: any) {
    const errorMsg = error?.message || String(error);
    if (errorMsg.includes("already exists") || errorMsg.includes("mutual exclusivity")) {
      console.log("  Authorization already exists on chain. Saving setup and continuing.");
      savePermAuthzSetup(authorityAddress, delegateAccount.address);
    } else {
      console.error(error);
      process.exit(1);
    }
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
