package journeys

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana/x/cs/types"
	detypes "github.com/verana-labs/verana/x/de/types"
	permtypes "github.com/verana-labs/verana/x/pp/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunPermissionSpendEnforcementJourney implements Journey 312: AUTHZ-CHECK-1
// operator spend_limit enforcement (#324). It is SELF-CONTAINED: it
// depends only on Journey 301 (the Corporation + funded accounts) and builds its
// own ecosystem -> schema -> root permission. The root carries non-zero
// validation_fees so each StartParticipantOP commits a non-zero nominal amount
// (fees + trust deposit) that the operator's spend_limit is debited by.
//
// TEST 1 (positive): grant a generous spend_limit, run one StartParticipantOP,
// assert remaining_spend was debited below the granted limit.
// TEST 2 (negative): re-grant a 1uvna spend_limit (below one operation's cost)
// and assert the next StartParticipantOP is rejected with a spend-limit error.
//
// Period auto-reset of remaining_spend is covered by the DE keeper unit tests
// (TestOperatorAuthzPeriodRenewal); a live period-reset journey is intentionally
// out of scope here (the participant overlap rules make looping operations on one
// root awkward and add little over the unit coverage).
func RunPermissionSpendEnforcementJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 312: PP Operator spend_limit enforcement (AUTHZ-CHECK-1)")

	setup := lib.LoadJourneyResult("journey301")
	policyAddr := setup.GroupPolicyAddr
	operatorAddr := setup.OperatorAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	fmt.Printf("  Corporation: %s\n", policyAddr)
	fmt.Printf("  Operator:    %s\n", operatorAddr)

	const (
		// validation_fees on the root, in trust units. trust_unit_price is
		// 1_000_000 uvna, so this drives ~10 VNA of fees per StartParticipantOP
		// plus the trust deposit on top.
		rootValidationFees = uint64(10)
		largeSpendLimit    = int64(100_000_000) // 100 VNA, comfortably above one op
		tinySpendLimit     = int64(1)           // 1 uvna, below one op's cost
	)

	// Sibling journeys spend from the shared policy account, so top it up to
	// guarantee it can cover the positive operation's fees + deposit.
	fmt.Println("\n--- Step 0: Top up the corporation policy account ---")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, policyAddr, math.NewInt(200_000_000))
	waitForTx("policy top-up")

	baseMsgTypes := []string{
		"/verana.ec.v1.MsgCreateEcosystem",
		"/verana.cs.v1.MsgCreateCredentialSchema",
		"/verana.pp.v1.MsgCreateRootParticipant",
		"/verana.pp.v1.MsgStartParticipantOP",
		"/verana.pp.v1.MsgSetParticipantOPToValidated",
	}

	// =========================================================================
	// Setup: grant prereq authz (no spend_limit), then build ecosystem -> schema
	// -> root permission with non-zero validation_fees.
	// =========================================================================
	fmt.Println("\n--- Step 1: Grant operator prereq authz (no spend_limit) ---")
	if err := lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr, baseMsgTypes,
	); err != nil {
		return fmt.Errorf("step 1 failed: grant prereq authz: %w", err)
	}
	waitForTx("grant prereq authz")

	fmt.Println("\n--- Step 2: Create Ecosystem ---")
	trIDStr, err := lib.CreateEcosystemWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		lib.GenerateUniqueDID(client, ctx),
		"https://spend-enforcement-test.com/governance-framework.pdf",
		"sha384-MzNNbQTWCSUSi0bbz7dbua+RcENv7C6FvlmYJ1Y+I727HsPOHdzwELMYO9Mz68M26",
		"en",
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: create ecosystem: %w", err)
	}
	trID, _ := strconv.ParseUint(trIDStr, 10, 64)
	waitForTx("ecosystem")

	fmt.Println("\n--- Step 3: Create Credential Schema (GRANTOR validation modes) ---")
	csIDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, lib.GenerateSimpleSchema(trIDStr),
		cschema.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
		cschema.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_GRANTOR_VALIDATION_PROCESS,
	)
	if err != nil {
		return fmt.Errorf("step 3 failed: create schema: %w", err)
	}
	csID, _ := strconv.ParseUint(csIDStr, 10, 64)
	waitForTx("schema")

	fmt.Println("\n--- Step 4: Create Root Permission (validation_fees > 0) ---")
	rootEffectiveFrom := time.Now().Add(2 * time.Second)
	rootEffectiveUntil := rootEffectiveFrom.Add(360 * 24 * time.Hour)
	rootID, err := lib.CreateRootPermissionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, lib.GenerateUniqueDID(client, ctx),
		&rootEffectiveFrom, &rootEffectiveUntil, rootValidationFees, 0, 0,
	)
	if err != nil {
		return fmt.Errorf("step 4 failed: create root permission: %w", err)
	}
	fmt.Printf("  Root permission id: %d (validation_fees=%d)\n", rootID, rootValidationFees)
	waitForTx("root permission")

	// The root validator MUST be ACTIVE before it can validate a child. Block time
	// can lag wall-clock, so poll until it passes effective_from.
	fmt.Println("  Waiting for root to become active (block time > effective_from)...")
	rootActive := false
	for i := 0; i < 40; i++ {
		st, serr := client.Context().Client.Status(ctx)
		if serr == nil && st.SyncInfo.LatestBlockTime.After(rootEffectiveFrom) {
			rootActive = true
			fmt.Printf("  Root active at block time %s\n", st.SyncInfo.LatestBlockTime)
			break
		}
		time.Sleep(2 * time.Second)
	}
	if !rootActive {
		return fmt.Errorf("step 4 failed: root did not become active in time")
	}

	// =========================================================================
	// TEST 1: positive path. Generous spend_limit -> operation succeeds and
	// remaining_spend is debited below the granted limit.
	// =========================================================================
	fmt.Println("\n=== TEST 1: spend_limit covers the operation (expect debit) ===")
	fmt.Printf("\n--- Step 5: Re-grant operator authz with spend_limit=%d uvna ---\n", largeSpendLimit)
	if err := lib.GrantOperatorAuthorizationWithSpendViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr, baseMsgTypes,
		sdk.NewCoins(sdk.NewCoin(permtypes.BondDenom, math.NewInt(largeSpendLimit))), nil,
	); err != nil {
		return fmt.Errorf("step 5 failed: grant authz with spend_limit: %w", err)
	}
	waitForTx("grant spend_limit (large)")

	fmt.Println("\n--- Step 6: StartParticipantOP (ISSUER_GRANTOR) within limit ---")
	childIDStr, err := lib.StartPermissionVPWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		permtypes.ParticipantRole_ISSUER_GRANTOR, rootID,
		lib.GenerateUniqueDID(client, ctx),
	)
	if err != nil {
		return fmt.Errorf("step 6 failed: StartParticipantOP within limit: %w", err)
	}
	fmt.Printf("OK Step 6: StartParticipantOP succeeded, child id %s\n", childIDStr)
	waitForTx("StartParticipantOP within limit")

	fmt.Println("\n--- Step 7: Assert remaining_spend was debited below the limit ---")
	remaining, err := queryOperatorRemainingSpend(ctx, client, policyAddr, operatorAddr)
	if err != nil {
		return fmt.Errorf("step 7 failed: query remaining_spend: %w", err)
	}
	if !remaining.IsPositive() {
		return fmt.Errorf("step 7 failed: expected positive remaining_spend, got %s", remaining)
	}
	if !remaining.LT(math.NewInt(largeSpendLimit)) {
		return fmt.Errorf("step 7 failed: remaining_spend %s was not debited below limit %d",
			remaining, largeSpendLimit)
	}
	fmt.Printf("OK Step 7: remaining_spend debited to %s uvna (< %d granted)\n", remaining, largeSpendLimit)

	// =========================================================================
	// TEST 2: negative path. A 1uvna spend_limit cannot cover one operation, so
	// the next StartParticipantOP MUST be rejected. A different role keeps it in a
	// distinct overlap context from the TEST 1 child.
	// =========================================================================
	fmt.Println("\n=== TEST 2: spend_limit below operation cost (expect rejection) ===")
	fmt.Printf("\n--- Step 8: Re-grant operator authz with spend_limit=%d uvna ---\n", tinySpendLimit)
	if err := lib.GrantOperatorAuthorizationWithSpendViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr, baseMsgTypes,
		sdk.NewCoins(sdk.NewCoin(permtypes.BondDenom, math.NewInt(tinySpendLimit))), nil,
	); err != nil {
		return fmt.Errorf("step 8 failed: grant authz with tiny spend_limit: %w", err)
	}
	waitForTx("grant spend_limit (tiny)")

	fmt.Println("\n--- Step 9: StartParticipantOP (VERIFIER_GRANTOR) exceeding limit (expect rejection) ---")
	_, err = lib.StartPermissionVPWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		permtypes.ParticipantRole_VERIFIER_GRANTOR, rootID,
		lib.GenerateUniqueDID(client, ctx),
	)
	if err == nil {
		return fmt.Errorf("step 9 failed: expected spend-limit rejection but operation succeeded")
	}
	if !strings.Contains(err.Error(), "spend limit exceeded") {
		return fmt.Errorf("step 9 failed: expected a spend-limit error, got: %w", err)
	}
	fmt.Printf("OK Step 9: Correctly rejected over-limit operation: %v\n", err)

	// =========================================================================
	// TEST 3: RepayParticipantSlashedTrustDeposit also debits spend_limit
	// (spec v4-rc3 AUTHZ-CHECK-1 step 3 applies to every fund-committing message).
	// Build a fresh ECOSYSTEM schema + root + validated ISSUER child, slash it,
	// then prove a repay debits remaining_spend (positive) and is rejected over a
	// tiny limit (negative).
	// =========================================================================
	fmt.Println("\n=== TEST 3: RepayParticipantSlashedTrustDeposit debits spend_limit ===")
	test3MsgTypes := append(append([]string{}, baseMsgTypes...),
		"/verana.pp.v1.MsgSlashParticipantTrustDeposit",
		"/verana.pp.v1.MsgRepayParticipantSlashedTrustDeposit",
	)

	fmt.Println("\n--- Step 10: Prep (grant generous, build ECOSYSTEM schema+root+validated ISSUER child) ---")
	if err := lib.GrantOperatorAuthorizationWithSpendViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr, test3MsgTypes,
		sdk.NewCoins(sdk.NewCoin(permtypes.BondDenom, math.NewInt(largeSpendLimit))), nil,
	); err != nil {
		return fmt.Errorf("step 10 failed: grant prep authz: %w", err)
	}
	waitForTx("grant prep authz (test 3)")

	cs2IDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, lib.GenerateSimpleSchema(trIDStr),
		cschema.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
		cschema.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
	)
	if err != nil {
		return fmt.Errorf("step 10 failed: create ECOSYSTEM schema: %w", err)
	}
	cs2ID, _ := strconv.ParseUint(cs2IDStr, 10, 64)
	waitForTx("schema 2")

	root2EffectiveFrom := time.Now().Add(2 * time.Second)
	root2EffectiveUntil := root2EffectiveFrom.Add(360 * 24 * time.Hour)
	root2ID, err := lib.CreateRootPermissionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		cs2ID, lib.GenerateUniqueDID(client, ctx),
		&root2EffectiveFrom, &root2EffectiveUntil, rootValidationFees, 0, 0,
	)
	if err != nil {
		return fmt.Errorf("step 10 failed: create root2: %w", err)
	}
	waitForTx("root 2")
	root2Active := false
	for i := 0; i < 40; i++ {
		st, serr := client.Context().Client.Status(ctx)
		if serr == nil && st.SyncInfo.LatestBlockTime.After(root2EffectiveFrom) {
			root2Active = true
			break
		}
		time.Sleep(2 * time.Second)
	}
	if !root2Active {
		return fmt.Errorf("step 10 failed: root2 did not become active in time")
	}

	repayChildIDStr, err := lib.StartPermissionVPWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		permtypes.ParticipantRole_ISSUER, root2ID, lib.GenerateUniqueDID(client, ctx),
	)
	if err != nil {
		return fmt.Errorf("step 10 failed: start ISSUER child: %w", err)
	}
	repayChildID, _ := strconv.ParseUint(repayChildIDStr, 10, 64)
	waitForTx("start ISSUER child")
	if _, err := lib.SetPermissionVPToValidated(client, ctx, operatorAccount, permtypes.MsgSetParticipantOPToValidated{
		Corporation: policyAddr,
		Id:          repayChildID,
	}); err != nil {
		return fmt.Errorf("step 10 failed: validate ISSUER child: %w", err)
	}
	waitForTx("validate ISSUER child")

	childPerm, err := lib.GetParticipant(client, ctx, repayChildID)
	if err != nil {
		return fmt.Errorf("step 10 failed: query child: %w", err)
	}
	if childPerm.Deposit < 4 {
		return fmt.Errorf("step 10 failed: child deposit %d too small to slash+repay", childPerm.Deposit)
	}
	slashAmount := childPerm.Deposit / 2
	fmt.Printf("OK Step 10: child %d deposit=%d, will slash %d (repay is always the full slashed_deposit)\n", repayChildID, childPerm.Deposit, slashAmount)

	fmt.Println("\n--- Step 11: Slash the child trust deposit (slashing does not consume spend) ---")
	if err := lib.SlashPermissionTrustDeposit(client, ctx, operatorAccount, policyAddr, repayChildID, slashAmount, "journey 312 spend test slash"); err != nil {
		return fmt.Errorf("step 11 failed: slash child: %w", err)
	}
	waitForTx("slash child")

	fmt.Printf("\n--- Step 12: spend_limit=%d (below slashed_deposit=%d), repay rejected ---\n", tinySpendLimit, slashAmount)
	if err := lib.GrantOperatorAuthorizationWithSpendViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr, test3MsgTypes,
		sdk.NewCoins(sdk.NewCoin(permtypes.BondDenom, math.NewInt(tinySpendLimit))), nil,
	); err != nil {
		return fmt.Errorf("step 12 failed: grant tiny authz: %w", err)
	}
	waitForTx("grant spend_limit (tiny, repay)")
	if err := lib.RepayPermissionSlashedTrustDeposit(client, ctx, operatorAccount, policyAddr, repayChildID); err == nil {
		return fmt.Errorf("step 12 failed: expected spend-limit rejection but repay succeeded")
	} else if !strings.Contains(err.Error(), "spend limit exceeded") {
		return fmt.Errorf("step 12 failed: expected a spend-limit error, got: %w", err)
	}
	fmt.Println("OK Step 12: Correctly rejected over-limit repay")

	fmt.Printf("\n--- Step 13: spend_limit=%d, repay full slashed_deposit within limit (expect debit) ---\n", largeSpendLimit)
	if err := lib.GrantOperatorAuthorizationWithSpendViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr, test3MsgTypes,
		sdk.NewCoins(sdk.NewCoin(permtypes.BondDenom, math.NewInt(largeSpendLimit))), nil,
	); err != nil {
		return fmt.Errorf("step 13 failed: grant authz: %w", err)
	}
	waitForTx("grant spend_limit (repay)")
	if err := lib.RepayPermissionSlashedTrustDeposit(client, ctx, operatorAccount, policyAddr, repayChildID); err != nil {
		return fmt.Errorf("step 13 failed: repay within limit: %w", err)
	}
	waitForTx("repay within limit")
	remainingAfterRepay, err := queryOperatorRemainingSpend(ctx, client, policyAddr, operatorAddr)
	if err != nil {
		return fmt.Errorf("step 13 failed: query remaining_spend: %w", err)
	}
	wantRemaining := math.NewInt(largeSpendLimit).Sub(math.NewIntFromUint64(slashAmount))
	if !remainingAfterRepay.Equal(wantRemaining) {
		return fmt.Errorf("step 13 failed: remaining_spend %s, want %s (limit - slashed_deposit)", remainingAfterRepay, wantRemaining)
	}
	fmt.Printf("OK Step 13: repay debited remaining_spend to %s uvna (= %d - %d)\n", remainingAfterRepay, largeSpendLimit, slashAmount)

	fmt.Println("\n========================================")
	fmt.Println("Journey 312 completed successfully!")
	fmt.Println("Operator spend_limit enforced on StartParticipantOP + RepaySlashedTrustDeposit:")
	fmt.Println("  debited within limit, rejected over limit.")
	fmt.Println("========================================")
	return nil
}

// queryOperatorRemainingSpend reads the operator's OperatorAuthorization for the
// given corporation policy account and returns its remaining_spend in uvna.
func queryOperatorRemainingSpend(
	ctx context.Context, client cosmosclient.Client, policyAddr, operatorAddr string,
) (math.Int, error) {
	qc := detypes.NewQueryClient(client.Context())
	resp, err := qc.ListOperatorAuthorizations(ctx, &detypes.QueryListOperatorAuthorizationsRequest{
		Operator: operatorAddr,
	})
	if err != nil {
		return math.Int{}, err
	}
	for _, oa := range resp.OperatorAuthorizations {
		if oa.Operator == operatorAddr {
			return oa.RemainingSpend.AmountOf(permtypes.BondDenom), nil
		}
	}
	return math.Int{}, fmt.Errorf("no OperatorAuthorization found for operator %s", operatorAddr)
}
