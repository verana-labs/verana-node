package journeys

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	xrtypes "github.com/verana-labs/verana-node/x/xr/types"

	"github.com/verana-labs/verana-node/testharness/lib"
)

// RunXrUpdateExchangeRateJourney implements Journey 602: XR Update Exchange Rate (feeder)
// Tests MsgUpdateExchangeRate with feeder authorization (fail without auth, grant, succeed with auth).
// Depends on Journey 601 (exchange rate ID) and Journey 301 (group, feeder).
func RunXrUpdateExchangeRateJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 602: XR Update Exchange Rate with Feeder Authorization")

	// =========================================================================
	// Step 1: Load journey 601 and 301 results
	// =========================================================================
	fmt.Println("\n--- Step 1: Load journey results ---")

	setup601 := lib.LoadJourneyResult("journey601")
	setup301 := lib.LoadJourneyResult("journey301")

	exchangeRateID, err := strconv.ParseUint(setup601.ExchangeRateID, 10, 64)
	if err != nil {
		return fmt.Errorf("step 1 failed: could not parse exchange rate ID: %w", err)
	}

	policyAddr := setup301.GroupPolicyAddr
	feederAddr := setup301.DelegateAddr

	feederAccount := lib.GetAccount(client, permDelegateName)
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)

	fmt.Printf("  Exchange Rate ID: %d\n", exchangeRateID)
	fmt.Printf("  Group Policy:     %s\n", policyAddr)
	fmt.Printf("  Feeder:         %s\n", feederAddr)
	fmt.Println("✅ Step 1: Loaded journey results")

	// =========================================================================
	// Step 2: Test UpdateExchangeRate WITHOUT authorization (expect failure)
	// =========================================================================
	fmt.Println("\n--- Step 2: Feeder tries UpdateExchangeRate without auth (expect failure) ---")

	updateMsg := &xrtypes.MsgUpdateExchangeRate{
		Feeder: feederAddr,
		Id:     exchangeRateID,
		Rate:   "2000000",
	}

	txResp, err := client.BroadcastTx(ctx, feederAccount, updateMsg)
	if err == nil && txResp.TxResponse.Code != 0 {
		err = fmt.Errorf("transaction failed with code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	if err := expectAuthorizationError("Step 2", err); err != nil {
		return err
	}
	fmt.Println("✅ Step 2: UpdateExchangeRate correctly rejected without authorization")
	waitForTx("update exchange rate rejection")

	// =========================================================================
	// Step 3: Grant delegate authorization for MsgUpdateExchangeRate via group proposal
	// =========================================================================
	fmt.Println("\n--- Step 3: Grant delegate auth for MsgUpdateExchangeRate ---")

	err = lib.GrantDelegateAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account,
		policyAddr, feederAddr, feederAddr,
		nil, // XR messages have no MsgType
	)
	if err != nil {
		return fmt.Errorf("step 3 failed: %w", err)
	}
	fmt.Println("✅ Step 3: Granted MsgUpdateExchangeRate authorization")
	waitForTx("grant update exchange rate auth")

	// =========================================================================
	// Step 4: Test UpdateExchangeRate WITH authorization (expect success)
	// =========================================================================
	fmt.Println("\n--- Step 4: Feeder updates exchange rate with auth ---")

	newRate := "2000000"
	updateMsg = &xrtypes.MsgUpdateExchangeRate{
		Feeder: feederAddr,
		Id:     exchangeRateID,
		Rate:   newRate,
	}

	txResp, err = client.BroadcastTx(ctx, feederAccount, updateMsg)
	if err != nil {
		return fmt.Errorf("step 4 failed: %w", err)
	}
	if txResp.TxResponse.Code != 0 {
		return fmt.Errorf("step 4 failed with code %d: %s", txResp.TxResponse.Code, txResp.TxResponse.RawLog)
	}
	fmt.Printf("✅ Step 4: UpdateExchangeRate succeeded (new rate: %s)\n", newRate)
	waitForTx("update exchange rate success")

	// =========================================================================
	// Step 5: Query to verify rate was updated
	// =========================================================================
	fmt.Println("\n--- Step 5: Query exchange rate to verify update ---")

	xrQueryClient := xrtypes.NewQueryClient(client.Context())
	getResp, err := xrQueryClient.GetExchangeRate(ctx, &xrtypes.QueryGetExchangeRateRequest{
		Id: exchangeRateID,
	})
	if err != nil {
		return fmt.Errorf("step 5 failed: could not query exchange rate: %w", err)
	}

	if getResp.ExchangeRate.Rate != newRate {
		return fmt.Errorf("step 5 failed: expected rate=%s, got rate=%s", newRate, getResp.ExchangeRate.Rate)
	}

	fmt.Printf("  Rate:    %s (expected: %s)\n", getResp.ExchangeRate.Rate, newRate)
	fmt.Printf("  State:   %v\n", getResp.ExchangeRate.State)
	fmt.Printf("  Updated: %s\n", getResp.ExchangeRate.Updated.Format(time.RFC3339))
	fmt.Println("✅ Step 5: Exchange rate updated successfully")

	// =========================================================================
	// Save results
	// =========================================================================
	result := lib.JourneyResult{
		ExchangeRateID:  setup601.ExchangeRateID,
		GroupID:         setup301.GroupID,
		GroupPolicyAddr: policyAddr,
		DelegateAddr:    feederAddr,
	}
	lib.SaveJourneyResult("journey602", result)

	fmt.Println("\n========================================")
	fmt.Println("Journey 602 completed successfully!")
	fmt.Println("XR UpdateExchangeRate with Feeder Authorization tested:")
	fmt.Println("  - UpdateExchangeRate: unauthorized rejected")
	fmt.Println("  - UpdateExchangeRate: authorized succeeded")
	fmt.Println("  - Exchange rate updated and verified")
	fmt.Println("========================================")

	return nil
}
