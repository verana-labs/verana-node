package journeys

import (
	"context"
	"fmt"
	"strconv"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	xrtypes "github.com/verana-labs/verana/x/xr/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunXrRevokeExchangeRateAuthzJourney implements Journey 605: XR Revoke Exchange Rate Authorization (governance)
// Revokes the ExchangeRateAuthorization granted in Journey 604 via gov proposal [MOD-XR-MSG-5],
// then verifies it is gone from GetExchangeRate.authorizations [MOD-XR-QRY-1].
// Depends on Journey 601 (exchange rate ID), Journey 301 (operator), and Journey 604 (grant).
func RunXrRevokeExchangeRateAuthzJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 605: XR Revoke Exchange Rate Authorization via Governance")

	govModuleAddr := authtypes.NewModuleAddress("gov").String()
	coolusrAddr := lib.COOLUSER_ADDRESS
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)

	// =========================================================================
	// Step 1: Load journey results
	// =========================================================================
	fmt.Println("\n--- Step 1: Load journey results ---")

	setup601 := lib.LoadJourneyResult("journey601")
	setup301 := lib.LoadJourneyResult("journey301")

	exchangeRateID, err := strconv.ParseUint(setup601.ExchangeRateID, 10, 64)
	if err != nil {
		return fmt.Errorf("step 1 failed: could not parse exchange rate ID: %w", err)
	}
	operatorAddr := setup301.OperatorAddr

	fmt.Printf("  Exchange Rate ID: %d\n", exchangeRateID)
	fmt.Printf("  Operator:         %s\n", operatorAddr)
	fmt.Println("✅ Step 1: Loaded journey results")

	// =========================================================================
	// Step 2: Submit gov proposal to revoke the authorization
	// =========================================================================
	fmt.Println("\n--- Step 2: Submit RevokeExchangeRateAuthorization governance proposal ---")

	revokeMsg := &xrtypes.MsgRevokeExchangeRateAuthorization{
		Authority: govModuleAddr,
		XrId:      exchangeRateID,
		Operator:  operatorAddr,
	}

	proposalID, err := submitXrGovProposal(
		client, ctx, coolusrAddr, cooluser,
		revokeMsg,
		"Revoke Exchange Rate Authorization",
		fmt.Sprintf("Revoke operator %s authorization on exchange rate %d", operatorAddr, exchangeRateID),
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}
	fmt.Printf("✅ Step 2: Submitted RevokeExchangeRateAuthorization proposal (ID: %d)\n", proposalID)

	// =========================================================================
	// Step 3: Vote and pass the proposal
	// =========================================================================
	fmt.Println("\n--- Step 3: Vote and pass the proposal ---")

	if err := voteAndPassGovProposal(client, ctx, proposalID); err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	fmt.Println("✅ Step 3: RevokeExchangeRateAuthorization proposal passed")

	// =========================================================================
	// Step 4: Verify the authorization is gone
	// =========================================================================
	fmt.Println("\n--- Step 4: Query exchange rate to verify the authorization is removed ---")

	xrQueryClient := xrtypes.NewQueryClient(client.Context())
	getResp, err := xrQueryClient.GetExchangeRate(ctx, &xrtypes.QueryGetExchangeRateRequest{
		Id: exchangeRateID,
	})
	if err != nil {
		return fmt.Errorf("step 4 failed: could not query exchange rate: %w", err)
	}

	for _, a := range getResp.Authorizations {
		if a.Operator == operatorAddr {
			return fmt.Errorf("step 4 failed: authorization for operator %s still present on exchange rate %d", operatorAddr, exchangeRateID)
		}
	}

	fmt.Printf("  Authorizations: %d (operator %s removed)\n", len(getResp.Authorizations), operatorAddr)
	fmt.Println("✅ Step 4: Exchange rate authorization revoked and verified")

	fmt.Println("\n========================================")
	fmt.Println("Journey 605 completed successfully!")
	fmt.Println("XR RevokeExchangeRateAuthorization via Governance tested:")
	fmt.Println("  - Revoke proposal submitted and passed")
	fmt.Println("  - Authorization removal verified via GetExchangeRate")
	fmt.Println("========================================")

	return nil
}
