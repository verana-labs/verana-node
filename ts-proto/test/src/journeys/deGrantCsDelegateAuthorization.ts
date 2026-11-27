/**
 * Journey: DE Grant CS Delegate Authorization
 *
 * This script grants delegate authorization from a CS authority account to a
 * CS delegate account for all 3 CS message types (plus TR create, so the
 * delegate can create the Trust Registry needed for CS tests).
 *
 * Key insight: When delegate is empty in MsgGrantDelegateAuthorization,
 * the AUTHZ-CHECK is skipped — the authority acts alone and signs directly.
 *
 * Run order: test:de-grant-cs-auth → test:cs-create → test:cs-update → test:cs-archive
 *
 * Usage:
 *   npm run test:de-grant-cs-auth
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
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgGrantDelegateAuthorization } from "../../../src/codec/verana/de/v1/tx";
import { saveCsAuthzSetup } from "../helpers/journeyResults";
import { MsgType, msgTypeToJSON } from "../../../src/codec/verana/de/v1/msg_type";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

// Derivation path indexes for CS authority and delegate (separate from TR's 10/11)
const AUTHORITY_INDEX = 12;
const DELEGATE_INDEX = 13;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: DE Grant CS Delegate Authorization (TypeScript Client)");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Create authority and delegate wallets
  console.log("Step 1: Creating CS authority and delegate wallets...");
  const authorityWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, AUTHORITY_INDEX);
  const delegateWallet = await createAccountFromMnemonic(COOLUSER_MNEMONIC, DELEGATE_INDEX);
  const cooluserWallet = await createWallet(COOLUSER_MNEMONIC);

  const authorityAccount = await getAccountInfo(authorityWallet);
  const delegateAccount = await getAccountInfo(delegateWallet);
  const cooluserAccount = await getAccountInfo(cooluserWallet);

  console.log(`  Cooluser:     ${cooluserAccount.address}`);
  console.log(`  CS Authority: ${authorityAccount.address} (derivation index ${AUTHORITY_INDEX})`);
  console.log(`  CS Delegate:  ${delegateAccount.address} (derivation index ${DELEGATE_INDEX})`);
  console.log();

  // Step 2: Fund authority and delegate from cooluser
  console.log("Step 2: Funding CS authority and delegate accounts...");
  const fundAmount = "50000000uvna"; // 50 VNA

  const fundAuthResult = await fundAccount(
    COOLUSER_MNEMONIC,
    cooluserAccount.address,
    authorityAccount.address,
    fundAmount,
  );
  if (fundAuthResult.code !== 0) {
    console.log(`  ❌ Failed to fund CS authority: ${fundAuthResult.rawLog}`);
    process.exit(1);
  }
  console.log(`  ✓ Funded CS authority with ${fundAmount}`);

  // Wait for authority funding tx to confirm
  const queryClient = await createQueryClient();
  console.log("  ⏳ Waiting for CS authority funding tx to confirm...");
  for (let i = 0; i < 30; i++) {
    try {
      const tx = await queryClient.getTx(fundAuthResult.transactionHash);
      if (tx) {
        console.log(`  ✓ CS authority funding confirmed at block ${tx.height}`);
        break;
      }
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }

  // Wait for cooluser sequence to advance before second fund tx
  console.log("  ⏳ Waiting for cooluser sequence to advance...");
  for (let i = 0; i < 30; i++) {
    try {
      const seq = await queryClient.getSequence(cooluserAccount.address);
      if (seq.sequence >= 1) {
        console.log(`  ✓ Cooluser sequence: ${seq.sequence}`);
        break;
      }
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }

  const fundOpResult = await fundAccount(
    COOLUSER_MNEMONIC,
    cooluserAccount.address,
    delegateAccount.address,
    fundAmount,
  );
  if (fundOpResult.code !== 0) {
    console.log(`  ❌ Failed to fund CS delegate: ${fundOpResult.rawLog}`);
    process.exit(1);
  }
  console.log(`  ✓ Funded CS delegate with ${fundAmount}`);

  // Wait for delegate funding tx to confirm
  console.log("  ⏳ Waiting for CS delegate funding tx to confirm...");
  for (let i = 0; i < 30; i++) {
    try {
      const tx = await queryClient.getTx(fundOpResult.transactionHash);
      if (tx) {
        console.log(`  ✓ CS delegate funding confirmed at block ${tx.height}`);
        break;
      }
    } catch {}
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }

  // Verify authority account balance before proceeding
  console.log("  ⏳ Verifying CS authority balance...");
  for (let i = 0; i < 20; i++) {
    const balance = await queryClient.getBalance(authorityAccount.address, config.denom);
    if (BigInt(balance.amount) > 0) {
      console.log(`  ✓ CS authority balance: ${balance.amount}${balance.denom}`);
      break;
    }
    await new Promise((resolve) => setTimeout(resolve, 1000));
  }
  queryClient.disconnect();
  console.log();

  // Step 3: Authority grants delegate authorization for EC create + all 3 CS message types
  // (v4-rc2: EC replaces TR as the parent of credential schemas).
  console.log("Step 3: Granting CS delegate authorization...");

  const allCsMsgTypes = [
    MsgType.MSG_TYPE_CREATE_ECOSYSTEM,            // needed so delegate can create the EC for CS tests
    MsgType.MSG_TYPE_CREATE_CREDENTIAL_SCHEMA,
    MsgType.MSG_TYPE_UPDATE_CREDENTIAL_SCHEMA,
    MsgType.MSG_TYPE_ARCHIVE_CREDENTIAL_SCHEMA,
  ];

  console.log("  Message types being authorized:");
  for (const msgType of allCsMsgTypes) {
    console.log(`    - ${msgTypeToJSON(msgType)}`);
  }

  const client = await createSigningClient(authorityWallet);

  const msg = {
    typeUrl: typeUrls.MsgGrantDelegateAuthorization,
    value: MsgGrantDelegateAuthorization.fromPartial({
      corporation: authorityAccount.address,
      // signer == corporation policy_address — authority acts alone (AUTHZ-CHECK-1 skipped)
      signer: authorityAccount.address,
      grantee: delegateAccount.address,
      msgTypes: allCsMsgTypes,
    }),
  };

  try {
    const fee = await calculateFeeWithSimulation(
      client,
      authorityAccount.address,
      [msg],
      "Granting CS delegate authorization",
    );
    console.log(`  Calculated gas: ${fee.gas}, fee: ${fee.amount[0].amount}${fee.amount[0].denom}`);

    const result = await signAndBroadcastWithRetry(
      client,
      authorityAccount.address,
      [msg],
      fee,
      "Granting CS delegate authorization",
    );

    console.log();
    if (result.code === 0) {
      console.log("✅ SUCCESS! CS delegate authorization granted!");
      console.log("=".repeat(60));
      console.log(`  Transaction Hash: ${result.transactionHash}`);
      console.log(`  Block Height:     ${result.height}`);
      console.log(`  Gas Used:         ${result.gasUsed}/${result.gasWanted}`);

      // Print relevant events
      const events = result.events || [];
      for (const event of events) {
        if (event.type.includes("grant") || event.type.includes("delegate")) {
          console.log(`  Event: ${event.type}`);
          for (const attr of event.attributes) {
            console.log(`    ${attr.key}: ${attr.value}`);
          }
        }
      }

      // Save CS authz setup for CS journeys
      saveCsAuthzSetup(authorityAccount.address, delegateAccount.address);
      console.log();
      console.log("  💾 Saved CS authority and delegate addresses for CS journeys");
    } else {
      console.log("❌ FAILED! Transaction failed.");
      console.log(`  Error Code: ${result.code}`);
      console.log(`  Raw Log:    ${result.rawLog}`);
      process.exit(1);
    }
  } catch (error: any) {
    console.log("❌ ERROR! Transaction failed with exception:");
    console.error(error);
    process.exit(1);
  } finally {
    client.disconnect();
  }

  console.log();
  console.log("=".repeat(60));
}

main().catch((error: any) => {
  console.error("\n❌ Fatal error:", error.message || error);

  if (error.cause?.code === "ECONNREFUSED" || error.message?.includes("fetch failed")) {
    console.error("\n⚠️  Connection Error: Cannot connect to the blockchain.");
    console.error(`   Make sure the Verana blockchain is running at ${config.rpcEndpoint}`);
  }

  process.exit(1);
});
