package journeys

import (
	"context"
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/ignite/cli/v28/ignite/pkg/cosmosclient"

	"github.com/verana-labs/verana-node/testharness/lib"
	detypes "github.com/verana-labs/verana-node/x/de/types"
)

// RunDeFeegrantAllowanceJourney implements Journey 313: AUTHZ-CHECK-2 fee grant.
// Granting a delegate authz with a feegrant_spend_limit realizes a cosmos x/feegrant allowance
// (MOD-DE-MSG-1-4); revoking it removes the allowance (MOD-DE-MSG-2-4). Self-
// contained off journey301.
func RunDeFeegrantAllowanceJourney(ctx context.Context, client cosmosclient.Client) error {
	fmt.Println("Starting Journey 313: DE fee grant -> x/feegrant allowance (AUTHZ-CHECK-2)")

	setup := lib.LoadJourneyResult("journey301")
	policyAddr := setup.GroupPolicyAddr
	delegateAddr := setup.DelegateAddr
	adminAccount := lib.GetAccount(client, permGroupAdminName)
	member1Account := lib.GetAccount(client, permGroupMember1Name)
	fmt.Printf("  Corporation (granter): %s\n  Delegate (grantee): %s\n", policyAddr, delegateAddr)

	fgClient := feegrant.NewQueryClient(client.Context())
	queryAllowance := func() (*feegrant.Grant, error) {
		resp, err := fgClient.Allowance(ctx, &feegrant.QueryAllowanceRequest{Granter: policyAddr, Grantee: delegateAddr})
		if err != nil {
			return nil, err
		}
		return resp.Allowance, nil
	}

	// Step 1: grant delegate authz WITH a corporation fee grant.
	fmt.Println("\n--- Step 1: Grant delegate authz with feegrant_spend_limit=100 VNA ---")
	exp := time.Now().Add(24 * time.Hour)
	if err := lib.GrantDelegateAuthorizationWithFeegrantViaGroup(
		client, ctx, adminAccount, member1Account, policyAddr, delegateAddr,
		[]detypes.MsgType{detypes.MsgType_MSG_TYPE_START_ONBOARDING},
		sdk.NewCoins(sdk.NewInt64Coin("uvna", 100000000)), &exp,
	); err != nil {
		return fmt.Errorf("step 1 failed: grant with feegrant: %w", err)
	}
	waitForTx("grant with feegrant")

	// Step 2: the cosmos x/feegrant allowance MUST now exist.
	fmt.Println("\n--- Step 2: Assert x/feegrant allowance created ---")
	grant, err := queryAllowance()
	if err != nil || grant == nil {
		return fmt.Errorf("step 2 failed: expected x/feegrant allowance for %s->%s, got err=%v grant=%v", policyAddr, delegateAddr, err, grant)
	}
	fmt.Printf("OK Step 2: x/feegrant allowance exists (granter=%s grantee=%s type=%s)\n", grant.Granter, grant.Grantee, grant.Allowance.TypeUrl)

	// Step 3: re-grant without feegrant_spend_limit -> allowance MUST be revoked.
	fmt.Println("\n--- Step 3: Re-grant without feegrant_spend_limit (revokes the allowance) ---")
	if err := lib.GrantDelegateAuthorizationViaGroup(
		client, ctx, adminAccount, member1Account, policyAddr, delegateAddr, delegateAddr,
		[]detypes.MsgType{detypes.MsgType_MSG_TYPE_START_ONBOARDING},
	); err != nil {
		return fmt.Errorf("step 3 failed: re-grant without feegrant: %w", err)
	}
	waitForTx("re-grant without feegrant")

	fmt.Println("\n--- Step 4: Assert x/feegrant allowance revoked ---")
	grant, err = queryAllowance()
	if err == nil && grant != nil {
		return fmt.Errorf("step 4 failed: expected allowance to be revoked, but it still exists")
	}
	fmt.Println("OK Step 4: x/feegrant allowance correctly revoked")

	fmt.Println("\n========================================")
	fmt.Println("Journey 313 completed successfully!")
	fmt.Println("DE fee grant realized + revoked the cosmos x/feegrant allowance (AUTHZ-CHECK-2).")
	fmt.Println("========================================")
	return nil
}
