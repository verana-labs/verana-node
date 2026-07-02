package journeys

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/google/uuid"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cschema "github.com/verana-labs/verana-node/x/cs/types"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	permtypes "github.com/verana-labs/verana-node/x/pp/types"

	"github.com/verana-labs/verana-node/testharness/lib"
)

// RunPermissionRecordEnforcementJourney implements Journey 314: AUTHZ-CHECK-3
// (VSOA record spend_limit) and AUTHZ-CHECK-4 (VSOA per-record fee_spend_limit)
// enforcement inside CreateOrUpdateParticipantSession (CSPS), #324.
//
// The record (VSOA) limits can only be set at participant creation via
// StartParticipantOP's --vs-operator-authz-* flags, so the fee-bearing issuers
// that carry the limits are built here in Go. The CSPS enforcement itself is
// driven two ways:
//   - CHECK-3 (record spend_limit): the vs_operator runs CSPS paying its own
//     fee; the record's remaining_spend is debited by trust_fees. Tested here.
//   - CHECK-4 (record fee_spend_limit): only triggers when the corp pays the tx
//     fee (fee_granter == corp). That needs a fee-granted tx, which is exercised
//     against issuer-X via the veranad CLI (see AUTHZ_FEE_LIVE_CLI_TEST.md). This
//     journey prints issuer-X + agent ids for that CLI step.
//
// Depends on Journey 302 (Corporation + ecosystem). Self-contained otherwise.
func RunPermissionRecordEnforcementJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 314: PP record spend/fee enforcement (AUTHZ-CHECK-3 / CHECK-4)")

	setup := lib.LoadJourneyResult("journey302")
	policyAddr := setup.GroupPolicyAddr
	operatorAddr := setup.OperatorAddr
	operatorAccount := lib.GetAccount(client, permOperatorName)
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)
	trID, _ := strconv.ParseUint(setup.EcosystemID, 10, 64)

	vsOperatorAccount := lib.GetAccount(client, "cooluser")
	vsOperatorAddr, _ := vsOperatorAccount.Address("verana")

	fmt.Printf("  Corporation: %s\n", policyAddr)
	fmt.Printf("  Operator:    %s\n", operatorAddr)
	fmt.Printf("  VS operator: %s\n", vsOperatorAddr)

	const (
		// issuance_fees on each root, in trust units. trust_unit_price ~1e6 uvna,
		// so each CSPS commits a non-zero trust_fees the record spend_limit is
		// debited by.
		rootIssuanceFees = uint64(1)
		bigSpendLimit    = int64(1_000_000_000) // 1000 VNA, never blocks
		feeSpendLimit    = int64(3_000_000)     // CHECK-4 cap, drained by CLI fees
		tinySpendLimit   = int64(1)             // 1 uvna, below one CSPS cost
	)
	cspsMsg := permtypes.MsgCreateOrUpdateParticipantSessionTypeURL

	fmt.Println("\n--- Step 0: Top up the corporation policy account (covers trust_fees) ---")
	lib.SendFunds(client, ctx, lib.COOLUSER_ADDRESS, policyAddr, math.NewInt(300_000_000))
	waitForTx("policy top-up")

	fmt.Println("\n--- Step 1: Grant operator prereq authz (CS, root, startop, validate) ---")
	baseMsgTypes := []string{
		"/verana.cs.v1.MsgCreateCredentialSchema",
		"/verana.pp.v1.MsgCreateRootParticipant",
		"/verana.pp.v1.MsgStartParticipantOP",
		"/verana.pp.v1.MsgSetParticipantOPToValidated",
	}
	if err := lib.GrantOperatorAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, operatorAddr, operatorAddr, baseMsgTypes,
	); err != nil {
		return fmt.Errorf("step 1 failed: grant prereq authz: %w", err)
	}
	waitForTx("grant prereq authz")

	// Build the shared agent participant (its own schema, to avoid overlap).
	agentPermID, err := buildValidatedIssuer(
		client, ctx, operatorAccount, policyAddr, trID, rootIssuanceFees,
		"", nil, false, nil, // no vs_operator on the agent
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: build agent participant: %w", err)
	}
	fmt.Printf("OK Step 2: agent participant id=%d\n", agentPermID)

	// Issuer-X: large spend_limit (never blocks) + finite fee_spend_limit +
	// with_feegrant. Exercises CHECK-3 positive (here) and CHECK-4 (CLI).
	issuerX, err := buildValidatedIssuer(
		client, ctx, operatorAccount, policyAddr, trID, rootIssuanceFees,
		vsOperatorAddr, []string{cspsMsg}, true,
		&recordLimits{spend: bigSpendLimit, feeSpend: feeSpendLimit},
	)
	if err != nil {
		return fmt.Errorf("step 3 failed: build issuer-X: %w", err)
	}
	fmt.Printf("OK Step 3: issuer-X id=%d (spend_limit=%d, fee_spend_limit=%d, with_feegrant=true)\n",
		issuerX, bigSpendLimit, feeSpendLimit)

	// Issuer-Y: tiny spend_limit so the very first CSPS is rejected on CHECK-3.
	issuerY, err := buildValidatedIssuer(
		client, ctx, operatorAccount, policyAddr, trID, rootIssuanceFees,
		vsOperatorAddr, []string{cspsMsg}, false,
		&recordLimits{spend: tinySpendLimit},
	)
	if err != nil {
		return fmt.Errorf("step 4 failed: build issuer-Y: %w", err)
	}
	fmt.Printf("OK Step 4: issuer-Y id=%d (spend_limit=%d uvna)\n", issuerY, tinySpendLimit)

	// =========================================================================
	// CHECK-3 positive: vs_operator runs CSPS for issuer-X -> record
	// remaining_spend is debited below the granted spend_limit.
	// =========================================================================
	fmt.Println("\n=== CHECK-3 positive: CSPS debits the record spend_limit ===")
	beforeSpend, _, ok := queryVsoaRemaining(ctx, client, vsOperatorAddr, issuerX)
	if !ok {
		return fmt.Errorf("check-3 failed: no VSOA record for issuer-X %d", issuerX)
	}
	if err := lib.CreatePermissionSession(
		client, ctx, vsOperatorAccount, policyAddr,
		uuid.New().String(), issuerX, 0, agentPermID, agentPermID,
	); err != nil {
		return fmt.Errorf("check-3 failed: CSPS within limit: %w", err)
	}
	waitForTx("CSPS issuer-X (CHECK-3)")
	afterSpend, _, _ := queryVsoaRemaining(ctx, client, vsOperatorAddr, issuerX)
	if !afterSpend.LT(beforeSpend) {
		return fmt.Errorf("check-3 failed: remaining_spend not debited (before=%s after=%s)", beforeSpend, afterSpend)
	}
	fmt.Printf("OK CHECK-3 positive: remaining_spend %s -> %s (debited %s)\n",
		beforeSpend, afterSpend, beforeSpend.Sub(afterSpend))

	// =========================================================================
	// CHECK-3 negative: issuer-Y has a 1uvna spend_limit, below one CSPS cost,
	// so the CSPS MUST be rejected with a spend-limit error.
	// =========================================================================
	fmt.Println("\n=== CHECK-3 negative: CSPS over the record spend_limit is rejected ===")
	err = lib.CreatePermissionSession(
		client, ctx, vsOperatorAccount, policyAddr,
		uuid.New().String(), issuerY, 0, agentPermID, agentPermID,
	)
	if err == nil {
		return fmt.Errorf("check-3 negative failed: expected rejection but CSPS succeeded")
	}
	if !strings.Contains(err.Error(), "spend limit exceeded") {
		return fmt.Errorf("check-3 negative failed: expected spend-limit error, got: %w", err)
	}
	fmt.Printf("OK CHECK-3 negative: correctly rejected over-limit CSPS: %v\n", err)

	fmt.Println("\n========================================")
	fmt.Println("Journey 314: CHECK-3 verified (record spend_limit debited + over-limit rejected).")
	fmt.Println("For CHECK-4 (per-record fee_spend_limit, corp pays the fee), run the fee-granted")
	fmt.Println("CSPS via the veranad CLI against issuer-X:")
	fmt.Printf("  vs_operator (signer) : %s\n", vsOperatorAddr)
	fmt.Printf("  corporation          : %s\n", policyAddr)
	fmt.Printf("  issuer-X id          : %d  (fee_spend_limit=%d)\n", issuerX, feeSpendLimit)
	fmt.Printf("  agent id             : %d\n", agentPermID)
	fmt.Println("========================================")
	return nil
}

// recordLimits carries the optional VSOA record limits for buildValidatedIssuer.
type recordLimits struct {
	spend    int64
	feeSpend int64
}

// buildValidatedIssuer creates a credential schema, a root permission with the
// given issuance_fees, an ISSUER participant under it (optionally with a
// vs_operator authorization record carrying limits), and validates it. The
// validation sets effective_until so any VSOA record is activated. Returns the
// issuer participant id.
func buildValidatedIssuer(
	client cosmosclient.Client,
	ctx context.Context,
	operatorAccount cosmosaccount.Account,
	policyAddr string,
	trID uint64,
	issuanceFees uint64,
	vsOperatorAddr string,
	vsMsgTypes []string,
	withFeegrant bool,
	limits *recordLimits,
) (uint64, error) {
	csIDStr, err := lib.CreateCredentialSchemaWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		trID, lib.GenerateSimpleSchema(strconv.FormatUint(trID, 10)),
		cschema.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
		cschema.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_ECOSYSTEM_VALIDATION_PROCESS,
	)
	if err != nil {
		return 0, fmt.Errorf("create schema: %w", err)
	}
	csID, _ := strconv.ParseUint(csIDStr, 10, 64)
	waitForTx("schema")

	effFrom := time.Now().Add(3 * time.Second)
	effUntil := effFrom.Add(360 * 24 * time.Hour)
	rootID, err := lib.CreateRootPermissionWithAuthority(
		client, ctx, operatorAccount, policyAddr,
		csID, lib.GenerateUniqueDID(client, ctx), &effFrom, &effUntil,
		0, issuanceFees, 0,
	)
	if err != nil {
		return 0, fmt.Errorf("create root: %w", err)
	}
	waitForTx("root")
	if err := waitRootActive(ctx, client, effFrom); err != nil {
		return 0, err
	}

	msg := permtypes.MsgStartParticipantOP{
		Corporation:            policyAddr,
		Role:                   permtypes.ParticipantRole_ISSUER,
		ValidatorParticipantId: rootID,
		Did:                    lib.GenerateUniqueDID(client, ctx),
	}
	if vsOperatorAddr != "" {
		msg.VsOperator = vsOperatorAddr
		msg.VsOperatorAuthzMsgTypes = vsMsgTypes
		msg.VsOperatorAuthzWithFeegrant = withFeegrant
		if limits != nil {
			if limits.spend > 0 {
				msg.VsOperatorAuthzSpendLimit = sdk.NewCoins(sdk.NewCoin(permtypes.BondDenom, math.NewInt(limits.spend)))
			}
			if limits.feeSpend > 0 {
				msg.VsOperatorAuthzFeeSpendLimit = sdk.NewCoins(sdk.NewCoin(permtypes.BondDenom, math.NewInt(limits.feeSpend)))
			}
		}
	}
	issuerIDStr, err := lib.StartPermissionVP(client, ctx, operatorAccount, msg)
	if err != nil {
		return 0, fmt.Errorf("start issuer VP: %w", err)
	}
	issuerID, _ := strconv.ParseUint(issuerIDStr, 10, 64)
	waitForTx("start issuer")

	valEffUntil := time.Now().Add(365 * 24 * time.Hour)
	if _, err := lib.SetPermissionVPToValidated(client, ctx, operatorAccount, permtypes.MsgSetParticipantOPToValidated{
		Corporation:    policyAddr,
		Id:             issuerID,
		EffectiveUntil: &valEffUntil,
	}); err != nil {
		return 0, fmt.Errorf("validate issuer: %w", err)
	}
	waitForTx("validate issuer")
	return issuerID, nil
}

// waitRootActive blocks until the chain block time passes effectiveFrom so the
// root can validate children.
func waitRootActive(ctx context.Context, client cosmosclient.Client, effectiveFrom time.Time) error {
	for i := 0; i < 40; i++ {
		st, err := client.Context().Client.Status(ctx)
		if err == nil && st.SyncInfo.LatestBlockTime.After(effectiveFrom) {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("root did not become active in time")
}

// queryVsoaRemaining returns the (remaining_spend, remaining_fee_spend) for the
// vs_operator's VSOA record on the given participant, in uvna.
func queryVsoaRemaining(
	ctx context.Context, client cosmosclient.Client, vsOperatorAddr string, participantID uint64,
) (math.Int, math.Int, bool) {
	qc := detypes.NewQueryClient(client.Context())
	resp, err := qc.ListVSOperatorAuthorizations(ctx, &detypes.QueryListVSOperatorAuthorizationsRequest{
		VsOperator: vsOperatorAddr,
	})
	if err != nil {
		return math.ZeroInt(), math.ZeroInt(), false
	}
	for _, vsoa := range resp.VsOperatorAuthorizations {
		for _, rec := range vsoa.Records {
			if rec.ParticipantId == participantID {
				return rec.RemainingSpend.AmountOf(permtypes.BondDenom),
					rec.RemainingFeeSpend.AmountOf(permtypes.BondDenom), true
			}
		}
	}
	return math.ZeroInt(), math.ZeroInt(), false
}
