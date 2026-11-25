/**
 * Journey: TD Reclaim Trust Deposit Yield
 *
 * MOD-TD-MSG-2 is void in v5: the message must be refused.
 *
 * Requires: test:de-grant-perm-auth must be run first (provides authority + operator).
 *
 * Usage:
 *   npm run test:td-reclaim-yield
 */

import {
  createDirectAccountFromMnemonic,
  createSigningClient,
  getAccountInfo,
  calculateFeeWithSimulation,
  signAndBroadcastWithRetry,
  config,
} from "../helpers/client";
import { typeUrls } from "../helpers/registry";
import { MsgReclaimTrustDepositYield } from "../../../src/codec/verana/td/v1/tx";
import { getPermAuthzSetup } from "../helpers/journeyResults";

const COOLUSER_MNEMONIC =
  (process.env.MNEMONIC && process.env.MNEMONIC.trim()) ||
  "pink glory help gown abstract eight nice crazy forward ketchup skill cheese";

const OPERATOR_INDEX = 15;

async function main() {
  console.log("=".repeat(60));
  console.log("Journey: TD Reclaim Trust Deposit Yield");
  console.log("=".repeat(60));
  console.log();

  // Step 1: Load DE auth setup
  console.log("Step 1: Loading DE authz setup...");
  const setup = getPermAuthzSetup();
  if (!setup) {
    console.log("  No DE authz setup found. Run test:de-grant-perm-auth first.");
    process.exit(1);
  }
  console.log(`  Authority: ${setup.authorityAddress}`);
  console.log(`  Operator:  ${setup.operatorAddress}`);
  console.log();

  // Step 2: Connect operator wallet
  console.log("Step 2: Setting up operator wallet...");
  const wallet = await createDirectAccountFromMnemonic(COOLUSER_MNEMONIC, OPERATOR_INDEX);
  const account = await getAccountInfo(wallet);
  const client = await createSigningClient(wallet);
  console.log(`  Connected as ${account.address}`);
  console.log();

  try {
    // Step 3: Send MsgReclaimTrustDepositYield and expect it to be refused
    console.log("Step 3: Sending MsgReclaimTrustDepositYield (void in v5, expect refusal)...");
    const msg = {
      typeUrl: typeUrls.MsgReclaimTrustDepositYield,
      value: MsgReclaimTrustDepositYield.fromPartial({
        corporation: setup.authorityAddress,
        operator: setup.operatorAddress,
      }),
    };

    let refusal = "";
    try {
      const fee = await calculateFeeWithSimulation(client, account.address, [msg], "Reclaiming trust deposit yield");
      const result = await signAndBroadcastWithRetry(client, account.address, [msg], fee, "Reclaiming trust deposit yield");
      refusal = result.code !== 0 ? result.rawLog || "" : "";
    } catch (error: any) {
      refusal = error?.message || String(error);
    }
    if (!refusal.includes("void in v5")) {
      throw new Error(`MsgReclaimTrustDepositYield was not refused as void: ${refusal || "tx succeeded"}`);
    }
    console.log();
    console.log("OK: MsgReclaimTrustDepositYield refused (void in v5).");
  } catch (error: any) {
    console.log("ERROR!");
    console.error(error);
    process.exit(1);
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
