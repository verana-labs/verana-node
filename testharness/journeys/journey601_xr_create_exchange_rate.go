package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/cosmos/cosmos-sdk/x/group"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	xrtypes "github.com/verana-labs/verana-node/x/xr/types"

	"github.com/verana-labs/verana-node/testharness/lib"
)

// RunXrCreateExchangeRateJourney implements Journey 601: XR Create Exchange Rate (council)
// Creates an exchange rate via council proposal, toggles state to true, then shows that a
// proposal below the 2/3 threshold is rejected and leaves state untouched.
func RunXrCreateExchangeRateJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 601: XR Create Exchange Rate via Council")

	council := lib.CouncilAuthority()
	cooluser := lib.GetAccount(client, lib.COOLUSER_NAME)

	fmt.Printf("  Council:     %s\n", council)
	fmt.Printf("  Proposer:    %s\n", lib.COOLUSER_ADDRESS)

	// =========================================================================
	// Step 1: Submit and pass a CreateExchangeRate proposal
	// =========================================================================
	fmt.Println("\n--- Step 1: CreateExchangeRate council proposal ---")

	createMsg := &xrtypes.MsgCreateExchangeRate{
		Authority:        council,
		BaseAssetType:    cstypes.PricingAssetType_TU,
		BaseAsset:        "tu",
		QuoteAssetType:   cstypes.PricingAssetType_COIN,
		QuoteAsset:       "uvna",
		Rate:             "1000000",
		RateScale:        6,
		ValidityDuration: 24 * time.Hour,
	}
	proposalID, err := lib.SubmitCouncilProposal(client, ctx, cooluser,
		"Create TU/uvna Exchange Rate",
		"Create exchange rate for TU to uvna conversion with rate=1000000, scale=6, validity=24h",
		createMsg)
	if err != nil {
		return fmt.Errorf("step 1 failed: %w", err)
	}
	if err := lib.PassCouncilProposal(client, ctx, proposalID); err != nil {
		return fmt.Errorf("step 1 failed: %w", err)
	}
	fmt.Println("✅ Step 1: CreateExchangeRate proposal executed")

	// =========================================================================
	// Step 2: Query the exchange rate to verify it was created with state=false
	// =========================================================================
	fmt.Println("\n--- Step 2: Query exchange rate to verify creation ---")

	xrQueryClient := xrtypes.NewQueryClient(client.Context())
	listResp, err := xrQueryClient.ListExchangeRates(ctx, &xrtypes.QueryListExchangeRatesRequest{
		BaseAssetType:  cstypes.PricingAssetType_TU,
		QuoteAssetType: cstypes.PricingAssetType_COIN,
		QuoteAsset:     "uvna",
	})
	if err != nil {
		return fmt.Errorf("step 2 failed: could not list exchange rates: %w", err)
	}
	if len(listResp.ExchangeRates) == 0 {
		return fmt.Errorf("step 2 failed: no exchange rates found for TU/uvna")
	}
	xr := listResp.ExchangeRates[len(listResp.ExchangeRates)-1]
	exchangeRateID := xr.Id
	fmt.Printf("  Exchange Rate ID: %d, state: %v\n", exchangeRateID, xr.State)
	if xr.State {
		return fmt.Errorf("step 2 failed: expected state=false, got state=true")
	}
	fmt.Println("✅ Step 2: Exchange rate created with state=false")

	// =========================================================================
	// Step 3: Submit and pass a SetExchangeRateState(true) proposal
	// =========================================================================
	fmt.Println("\n--- Step 3: SetExchangeRateState(true) council proposal ---")

	toggleID, err := lib.SubmitCouncilProposal(client, ctx, cooluser,
		"Activate TU/uvna Exchange Rate", "Set exchange rate state to true",
		&xrtypes.MsgSetExchangeRateState{Authority: council, Id: exchangeRateID, State: true})
	if err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	if err := lib.PassCouncilProposal(client, ctx, toggleID); err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	getResp, err := xrQueryClient.GetExchangeRate(ctx, &xrtypes.QueryGetExchangeRateRequest{Id: exchangeRateID})
	if err != nil {
		return fmt.Errorf("step 3 failed: could not query exchange rate: %w", err)
	}
	if !getResp.ExchangeRate.State {
		return fmt.Errorf("step 3 failed: expected state=true, got state=false")
	}
	fmt.Println("✅ Step 3: Exchange rate state is now true")

	// =========================================================================
	// Step 4: A 1-of-3 yes vote must be rejected at the end of the ballot
	// =========================================================================
	fmt.Println("\n--- Step 4: Below-threshold proposal is rejected ---")

	rejectID, err := lib.SubmitCouncilProposal(client, ctx, cooluser,
		"Deactivate TU/uvna Exchange Rate", "Only one of three seats votes yes",
		&xrtypes.MsgSetExchangeRateState{Authority: council, Id: exchangeRateID, State: false})
	if err != nil {
		return fmt.Errorf("step 4 failed: %w", err)
	}
	if err := lib.VoteCouncilProposal(client, ctx, cooluser, rejectID, true); err != nil {
		return fmt.Errorf("step 4 failed: %w", err)
	}
	fmt.Println("    - Waiting for the 30s ballot to close...")
	time.Sleep(35 * time.Second)
	proposal, err := lib.QueryCouncilProposal(client, ctx, rejectID)
	if err != nil {
		return fmt.Errorf("step 4 failed: %w", err)
	}
	if proposal == nil || proposal.Status != group.PROPOSAL_STATUS_REJECTED {
		return fmt.Errorf("step 4 failed: expected REJECTED, got %v", proposal)
	}
	getResp, err = xrQueryClient.GetExchangeRate(ctx, &xrtypes.QueryGetExchangeRateRequest{Id: exchangeRateID})
	if err != nil {
		return fmt.Errorf("step 4 failed: could not query exchange rate: %w", err)
	}
	if !getResp.ExchangeRate.State {
		return fmt.Errorf("step 4 failed: rejected proposal changed state")
	}
	fmt.Println("✅ Step 4: 1-of-3 proposal rejected, state unchanged")

	lib.SaveJourneyResult("journey601", lib.JourneyResult{ExchangeRateID: strconv.FormatUint(exchangeRateID, 10)})

	fmt.Println("\n========================================")
	fmt.Println("Journey 601 completed successfully!")
	fmt.Println("XR CreateExchangeRate via Council tested:")
	fmt.Println("  - CreateExchangeRate: proposal passed 2 of 3 and executed")
	fmt.Println("  - SetExchangeRateState(true): proposal passed and executed")
	fmt.Println("  - SetExchangeRateState(false): 1 of 3 rejected")
	fmt.Printf("  - Exchange Rate ID: %d\n", exchangeRateID)
	fmt.Println("========================================")
	return nil
}
