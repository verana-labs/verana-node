package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	xrtypes "github.com/verana-labs/verana/x/xr/types"

	"github.com/verana-labs/verana/testharness/lib"
)

// RunXrGrantExchangeRateAuthzJourney implements Journey 604: XR Grant Exchange Rate Authorization (governance)
// Grants an ExchangeRateAuthorization to an operator for a specific exchange rate via gov proposal [MOD-XR-MSG-4],
// then verifies it appears in GetExchangeRate.authorizations [MOD-XR-QRY-1].
// Depends on Journey 601 (exchange rate ID) and Journey 301 (operator).
func RunXrGrantExchangeRateAuthzJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 604: XR Grant Exchange Rate Authorization via Governance")

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
	// Step 2: Submit gov proposal to grant the authorization
	// =========================================================================
	fmt.Println("\n--- Step 2: Submit GrantExchangeRateAuthorization governance proposal ---")

	expiration := time.Now().Add(720 * time.Hour)
	grantMsg := &xrtypes.MsgGrantExchangeRateAuthorization{
		Authority:  govModuleAddr,
		XrId:       exchangeRateID,
		Operator:   operatorAddr,
		Expiration: &expiration,
	}

	proposalID, err := submitXrGovProposal(
		client, ctx, coolusrAddr, cooluser,
		grantMsg,
		"Grant Exchange Rate Authorization",
		fmt.Sprintf("Authorize operator %s to update exchange rate %d", operatorAddr, exchangeRateID),
	)
	if err != nil {
		return fmt.Errorf("step 2 failed: %w", err)
	}
	fmt.Printf("✅ Step 2: Submitted GrantExchangeRateAuthorization proposal (ID: %d)\n", proposalID)

	// =========================================================================
	// Step 3: Vote and pass the proposal
	// =========================================================================
	fmt.Println("\n--- Step 3: Vote and pass the proposal ---")

	if err := voteAndPassGovProposal(client, ctx, proposalID); err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	fmt.Println("✅ Step 3: GrantExchangeRateAuthorization proposal passed")

	// =========================================================================
	// Step 4: Verify the authorization is present
	// =========================================================================
	fmt.Println("\n--- Step 4: Query exchange rate to verify the authorization ---")

	xrQueryClient := xrtypes.NewQueryClient(client.Context())
	getResp, err := xrQueryClient.GetExchangeRate(ctx, &xrtypes.QueryGetExchangeRateRequest{
		Id: exchangeRateID,
	})
	if err != nil {
		return fmt.Errorf("step 4 failed: could not query exchange rate: %w", err)
	}

	found := false
	for _, a := range getResp.Authorizations {
		if a.Operator == operatorAddr {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("step 4 failed: authorization for operator %s not found on exchange rate %d", operatorAddr, exchangeRateID)
	}

	fmt.Printf("  Authorizations: %d (operator %s present)\n", len(getResp.Authorizations), operatorAddr)
	fmt.Println("✅ Step 4: Exchange rate authorization granted and verified")

	lib.SaveJourneyResult("journey604", lib.JourneyResult{
		ExchangeRateID: setup601.ExchangeRateID,
		OperatorAddr:   operatorAddr,
	})

	fmt.Println("\n========================================")
	fmt.Println("Journey 604 completed successfully!")
	fmt.Println("XR GrantExchangeRateAuthorization via Governance tested:")
	fmt.Println("  - Grant proposal submitted and passed")
	fmt.Println("  - Authorization verified via GetExchangeRate")
	fmt.Println("========================================")

	return nil
}
