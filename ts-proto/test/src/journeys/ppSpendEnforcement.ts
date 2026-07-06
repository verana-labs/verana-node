/**
 * Journey: PP Operator spend_limit enforcement (AUTHZ-CHECK-1, #324)
 *
 * Self-contained: creates fresh prerequisites (EC + CS + fee-bearing root), then:
 *   TEST 1 (positive): re-grant the operator a generous authz_spend_limit, run one
 *     MsgStartParticipantOP, and assert remaining_spend was debited below the limit.
 *   TEST 2 (negative): re-grant a 1uvna authz_spend_limit (below one operation's
 *     cost) and assert the next MsgStartParticipantOP is rejected with a spend error.
 *
 * The grant carries authz_spend_limit and is executed via the corporation's
 * x/group policy (MsgSubmitProposal + MsgVote EXEC_TRY), signed in
 * SIGN_MODE_LEGACY_AMINO_JSON. Mirrors the Go harness journey 312.
 *
 * Requires: test:de-grant-perm-auth first (provides the corporation + operator).
 *
 * Usage:
 *   npm run test:perm-spend-enforcement
 */

import {
  createAccountFromMnemonic,
  createWallet,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  waitForPermissionToBecomeEffective,
  createQueryClient,
  generateUniqueDID,
  fundAccount,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { IssuerOnboardingMode } from "../../../src/codec/verana/cs/v1/types";
import { ParticipantRole, OptionalUInt64 } from "../../../src/codec/verana/pp/v1/types";
import { MsgStartParticipantOP } from "../../../src/codec/verana/pp/v1/tx";
import { MsgGrantOperatorAuthorization } from "../../../src/codec/verana/de/v1/tx";
import { QueryClientImpl } from "../../../src/codec/verana/de/v1/query";
import { getPermAuthzSetup } from "../helpers/journeyResults";
import { createPermPrerequisites, createRootPermWithOperator, createCSWithOperator } from "../helpers/permissionHelpers";
import { MsgSubmitProposal, MsgVote, Exec } from "cosmjs-types/cosmos/group/v1/tx";
import { VoteOption } from "cosmjs-types/cosmos/group/v1/types";
import { QueryClient, createProtobufRpcClient } from "@cosmjs/stargate";
import { connectComet } from "@cosmjs/tendermint-rpc";
import type { SigningStargateClient } from "@cosmjs/stargate";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;
const SIGNER_INDEX = 10; // corp group member (proposer/voter)

const LARGE_SPEND_LIMIT = "100000000"; // 100 VNA, comfortably above one operation
const TINY_SPEND_LIMIT = "1"; // 1 uvna, below one operation's cost

const GRANT_MSG_TYPES = [
  typeUrls.MsgCreateEcosystem,
  typeUrls.MsgCreateCredentialSchema,
  typeUrls.MsgCreateRootParticipant,
  typeUrls.MsgStartParticipantOP,
  typeUrls.MsgSetParticipantOPToValidated,
];

/**
 * Re-grants the operator authorization via the corporation's x/group policy, with
 * the given authz_spend_limit. Submits MsgSubmitProposal then votes YES + EXEC_TRY
 * so the policy (threshold=1, weight=1) executes the inner grant. Grants are
 * in-place upserts, so this resets remaining_spend to the new limit.
 */
async function reGrantWithSpendLimit(
  client: SigningStargateClient,
  signerAddress: string,
  corporation: string,
  grantee: string,
  spendLimitAmount: string,
): Promise<void> {
  const innerMsgValue = MsgGrantOperatorAuthorization.encode(
    MsgGrantOperatorAuthorization.fromPartial({
      corporation,
      operator: corporation,
      grantee,
      msgTypes: GRANT_MSG_TYPES,
      withFeegrant: false,
      authzSpendLimit: [{ denom: config.denom, amount: spendLimitAmount }],
    }),
  ).finish();

  const proposalMsg = {
    typeUrl: "/cosmos.group.v1.MsgSubmitProposal",
    value: MsgSubmitProposal.fromPartial({
      groupPolicyAddress: corporation,
      proposers: [signerAddress],
      metadata: "Grant operator authz with spend_limit",
      title: "Grant operator authz with spend_limit",
      summary: `Grant operator authz with authz_spend_limit=${spendLimitAmount}${config.denom}`,
      messages: [{ typeUrl: typeUrls.MsgGrantOperatorAuthorization, value: innerMsgValue }],
      exec: Exec.EXEC_UNSPECIFIED,
    }),
  };

  const proposalFee = await calculateFeeWithSimulation(client, signerAddress, [proposalMsg], "Submit spend grant proposal");
  const proposalResult = await signAndBroadcastWithRetry(client, signerAddress, [proposalMsg], proposalFee, "Submit spend grant proposal");
  if (proposalResult.code !== 0) {
    throw new Error(`Failed to submit grant proposal: ${proposalResult.rawLog}`);
  }
  let proposalId: bigint | undefined;
  for (const event of proposalResult.events || []) {
    for (const attr of event.attributes) {
      if (attr.key === "proposal_id") {
        proposalId = BigInt(String(attr.value).replace(/"/g, ""));
        break;
      }
    }
    if (proposalId !== undefined) break;
  }
  if (proposalId === undefined) throw new Error("Could not extract proposal_id from grant proposal");

  const qc = await createQueryClient();
  for (let i = 0; i < 30; i++) {
    try {
      if (await qc.getTx(proposalResult.transactionHash)) break;
    } catch {}
    await new Promise((r) => setTimeout(r, 1000));
  }
  qc.disconnect();

  const voteMsg = {
    typeUrl: "/cosmos.group.v1.MsgVote",
    value: MsgVote.fromPartial({ proposalId, voter: signerAddress, option: VoteOption.VOTE_OPTION_YES, exec: Exec.EXEC_TRY, metadata: "" }),
  };
  const voteFee = await calculateFeeWithSimulation(client, signerAddress, [voteMsg], "Vote YES spend grant");
  const voteResult = await signAndBroadcastWithRetry(client, signerAddress, [voteMsg], voteFee, "Vote YES spend grant");
  if (voteResult.code !== 0) {
    throw new Error(`Failed to vote on grant proposal: ${voteResult.rawLog}`);
  }

  const qc2 = await createQueryClient();
  for (let i = 0; i < 30; i++) {
    try {
      if (await qc2.getTx(voteResult.transactionHash)) break;
    } catch {}
    await new Promise((r) => setTimeout(r, 1000));
  }
  qc2.disconnect();
}

/** Reads the operator's remaining_spend (in config.denom) from the DE module. */
async function queryRemainingSpend(operator: string): Promise<bigint> {
  const cometClient = await connectComet(config.rpcEndpoint);
  try {
    const queryClient = new QueryClient(cometClient as any);
    const rpc = createProtobufRpcClient(queryClient);
    const de = new QueryClientImpl(rpc);
    const resp = await de.ListOperatorAuthorizations({ corporationId: 0, operator, responseMaxSize: 64 });
    for (const oa of resp.operatorAuthorizations) {
      if (oa.operator === operator) {
        const coin = oa.remainingSpend.find((c) => c.denom === config.denom);
        return coin ? BigInt(coin.amount) : 0n;
      }
    }
    throw new Error(`No OperatorAuthorization found for operator ${operator}`);
  } finally {
    cometClient.disconnect();
  }
}

/** Broadcasts a MsgStartParticipantOP, returning the tx code (or throwing on rejection). */
async function startParticipantOp(
  client: SigningStargateClient,
  signer: string,
  corporation: string,
  operator: string,
  role: ParticipantRole,
  rootPermId: number,
): Promise<void> {
  const msg = {
    typeUrl: typeUrls.MsgStartParticipantOP,
    value: MsgStartParticipantOP.fromPartial({
      corporation,
      operator,
      role,
      validatorParticipantId: rootPermId,
      did: generateUniqueDID(),
      validationFees: OptionalUInt64.fromPartial({ value: 5 }),
      issuanceFees: OptionalUInt64.fromPartial({ value: 5 }),
      verificationFees: OptionalUInt64.fromPartial({ value: 5 }),
      vsOperator: "",
    }),
  };
  const fee = await calculateFeeWithSimulation(client, signer, [msg], "StartParticipantOP");
  const result = await signAndBroadcastWithRetry(client, signer, [msg], fee, "StartParticipantOP");
  if (result.code !== 0) {
    throw new Error(`StartParticipantOP failed: ${result.rawLog}`);
  }
}

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: PP Operator spend_limit enforcement (AUTHZ-CHECK-1)");
  console.log("=".repeat(60));
  console.log();

  console.log("Step 1: Loading PERM authz setup...");
  const setup = getPermAuthzSetup();
  if (!setup) {
    console.log("  Missing setup. Run test:de-grant-perm-auth first.");
    process.exit(1);
  }
  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${setup.operatorAddress}`);
  console.log();

  console.log("Step 2: Setting up wallets (legacy amino)...");
  const operatorWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const operatorAccount = await getAccountInfo(operatorWallet);
  const operatorClient = await createSigningClient(operatorWallet);

  const signerWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, SIGNER_INDEX);
  const signerAccount = await getAccountInfo(signerWallet);
  const signerClient = await createSigningClient(signerWallet);
  console.log(`  Operator: ${operatorAccount.address}, Group signer: ${signerAccount.address}`);
  console.log();

  try {
    // Step 0: Top up the corporation policy account so it can cover the
    // operations' fees + trust deposits.
    console.log("Step 0: Topping up the corporation policy account...");
    const cooluserWallet = await createWallet(COOLUSER_MNEMONIC);
    const cooluserAccount = await getAccountInfo(cooluserWallet);
    const fundResult = await fundAccount(COOLUSER_MNEMONIC, cooluserAccount.address, setup.authorityAddress, "500000000uvna");
    if (fundResult.code !== 0) throw new Error(`Failed to fund policy: ${fundResult.rawLog}`);
    console.log("  Policy funded with 500 VNA");
    console.log();

    // Step 3: Fresh prerequisites. The roots carry non-zero validation_fees
    // (default 5), so each StartParticipantOP commits a non-zero nominal amount
    // (fees + trust deposit) that spend_limit is debited by. TEST 2 needs a
    // distinct overlap context from TEST 1; since the schema only supports ISSUER
    // and two ECOSYSTEM roots on one schema overlap, we build a SECOND schema with
    // its own root and start each ISSUER under its own (schema, root).
    console.log("Step 3: Creating fresh prerequisites (EC + 2x CS + 2x fee-bearing Root)...");
    const ECOSYSTEM_MODE = IssuerOnboardingMode.ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS;
    const { ecId, schemaId: schema1Id, rootPermId: root1Id } = await createPermPrerequisites(
      operatorClient,
      setup.authorityAddress,
      setup.operatorAddress,
      ECOSYSTEM_MODE,
    );
    const schema2Id = await createCSWithOperator(operatorClient, setup.authorityAddress, setup.operatorAddress, ecId, ECOSYSTEM_MODE);
    const { rootPermId: root2Id, effectiveFrom: effectiveFrom2 } = await createRootPermWithOperator(
      operatorClient,
      setup.authorityAddress,
      setup.operatorAddress,
      schema2Id,
      generateUniqueDID(),
    );
    console.log(`  EC ${ecId} | schema1 ${schema1Id}/root1 ${root1Id} | schema2 ${schema2Id}/root2 ${root2Id}`);

    console.log("Step 4: Waiting for both root permissions to become effective...");
    const queryClient = await createQueryClient();
    try {
      // effectiveFrom2 is the later of the two, so waiting on it covers both.
      await waitForPermissionToBecomeEffective(queryClient, effectiveFrom2, 60000);
    } finally {
      queryClient.disconnect();
    }
    console.log("  Root permissions are now effective");
    console.log();

    // =====================================================================
    // TEST 1: generous spend_limit -> operation succeeds, remaining debited.
    // =====================================================================
    console.log("=== TEST 1: spend_limit covers the operation (expect debit) ===");
    console.log(`Step 5: Re-grant operator authz with spend_limit=${LARGE_SPEND_LIMIT}${config.denom}...`);
    await reGrantWithSpendLimit(signerClient, signerAccount.address, setup.authorityAddress, setup.operatorAddress, LARGE_SPEND_LIMIT);

    console.log("Step 6: StartParticipantOP (ISSUER under root1) within limit...");
    await startParticipantOp(operatorClient, operatorAccount.address, setup.authorityAddress, setup.operatorAddress, ParticipantRole.ISSUER, root1Id);
    console.log("  StartParticipantOP succeeded");

    console.log("Step 7: Asserting remaining_spend was debited below the limit...");
    const remaining = await queryRemainingSpend(setup.operatorAddress);
    if (remaining <= 0n) throw new Error(`expected positive remaining_spend, got ${remaining}`);
    if (remaining >= BigInt(LARGE_SPEND_LIMIT)) {
      throw new Error(`remaining_spend ${remaining} was not debited below limit ${LARGE_SPEND_LIMIT}`);
    }
    console.log(`  remaining_spend debited to ${remaining}${config.denom} (< ${LARGE_SPEND_LIMIT} granted)`);
    console.log();

    // =====================================================================
    // TEST 2: 1uvna spend_limit -> operation rejected. A different role keeps it
    // in a distinct overlap context from the TEST 1 child.
    // =====================================================================
    console.log("=== TEST 2: spend_limit below operation cost (expect rejection) ===");
    console.log(`Step 8: Re-grant operator authz with spend_limit=${TINY_SPEND_LIMIT}${config.denom}...`);
    await reGrantWithSpendLimit(signerClient, signerAccount.address, setup.authorityAddress, setup.operatorAddress, TINY_SPEND_LIMIT);

    console.log("Step 9: StartParticipantOP (ISSUER under root2) exceeding limit (expect rejection)...");
    let rejected = false;
    try {
      await startParticipantOp(operatorClient, operatorAccount.address, setup.authorityAddress, setup.operatorAddress, ParticipantRole.ISSUER, root2Id);
    } catch (err: any) {
      const m = err?.message || String(err);
      if (!m.includes("spend limit exceeded")) throw new Error(`expected a spend-limit error, got: ${m}`);
      rejected = true;
      console.log(`  Correctly rejected over-limit operation: ${m.split("\n")[0]}`);
    }
    if (!rejected) throw new Error("expected spend-limit rejection but operation succeeded");
    console.log();

    console.log("=".repeat(60));
    console.log("SUCCESS! Operator spend_limit enforced: debited within limit, rejected over limit.");
    console.log("=".repeat(60));
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error?.message || error);
    process.exit(1);
  } finally {
    operatorClient.disconnect();
    signerClient.disconnect();
  }
}

main().catch((error: any) => {
  console.error("\nFatal error:", error.message || error);
  process.exit(1);
});
