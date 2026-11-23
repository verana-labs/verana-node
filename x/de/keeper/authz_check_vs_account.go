package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/de/types"
)

// CheckVSAccountAuthorization implements [AUTHZ-CHECK-3] steps 1 to 5 for signer
// acting for the corporation that AUTHZ-CHECK-5 resolved to corporationID, on
// the entry vs resolved by the calling method. perm is the primary entry, nil
// when the method takes a vs_id directly. The spend of step 6 is applied by
// ConsumeVSAccountSpend once the handler knows the amount.
func (k Keeper) CheckVSAccountAuthorization(
	ctx context.Context,
	corporationID uint64,
	signer string,
	perm *types.AccreditationView,
	vs *types.VerifiableServiceView,
	msgType types.MsgType,
) error {
	if perm != nil && !perm.Active {
		return errorsmod.Wrapf(types.ErrAccreditationNotActive, "accreditation %d", perm.Id)
	}
	if vs.CorporationId != corporationID || vs.VsAccount != signer {
		return errorsmod.Wrapf(types.ErrVSAccountNotAuthorized,
			"%s is not the VS account of entry %d of corporation %d", signer, vs.Id, corporationID)
	}
	if len(vs.MsgTypes) == 0 {
		return errorsmod.Wrapf(types.ErrVSAccountNotAuthorized, "entry %d authorizes no message type", vs.Id)
	}
	if !slices.Contains(vs.MsgTypes, msgType) {
		return fmt.Errorf("%w: %s", types.ErrAuthzMsgTypeNotFound, msgType)
	}

	// 5. Operation-budget cycle; never aborts.
	if vs.Period == nil {
		return nil
	}
	budget, found, err := k.getVsAccountBudget(ctx, vs.Id)
	if err != nil || !found {
		return err
	}
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	if budget.Expiration == nil || now.Before(*budget.Expiration) {
		return nil
	}
	budget.RemainingSpend = budget.SpendLimit
	next := now.Add(*vs.Period)
	budget.Expiration = &next
	return k.vsBudgetRef.S.Set(ctx, vs.Id, budget)
}

// ConsumeVSAccountSpend implements [AUTHZ-CHECK-3] step 6: when entry vsID has a
// budget, its remaining spend MUST cover amount, which is then deducted per
// denom. The entry itself is not written. Callers run
// CheckVSAccountAuthorization first, so the cycle is current.
func (k Keeper) ConsumeVSAccountSpend(ctx context.Context, vsID uint64, amount sdk.Coins) error {
	if amount.IsZero() {
		return nil
	}
	budget, found, err := k.getVsAccountBudget(ctx, vsID)
	if err != nil || !found {
		return err
	}
	if !budget.RemainingSpend.IsAllGTE(amount) {
		return fmt.Errorf("%w: spend %s exceeds remaining %s",
			types.ErrAuthzSpendLimitExceeded, amount.String(), budget.RemainingSpend.String())
	}
	budget.RemainingSpend = budget.RemainingSpend.Sub(amount...)
	return k.vsBudgetRef.S.Set(ctx, vsID, budget)
}

// CheckVSAccountFeeGrant implements [AUTHZ-CHECK-4]: when the transaction fee
// granter is the corporation, vs.fee_spend_limit MUST be non-empty. The fee is
// checked and deducted by the x/feegrant allowance, never here.
func (k Keeper) CheckVSAccountFeeGrant(ctx context.Context, corporation string, vs *types.VerifiableServiceView) error {
	granter := FeeGranterFromContext(sdk.UnwrapSDKContext(ctx))
	if len(granter) == 0 {
		return nil
	}
	corpAddr, err := sdk.AccAddressFromBech32(corporation)
	if err != nil {
		return err
	}
	if !bytes.Equal(granter, corpAddr) {
		return nil
	}
	if len(vs.FeeSpendLimit) == 0 {
		return errorsmod.Wrapf(types.ErrVSAccountFeeGrantNotEnabled, "entry %d", vs.Id)
	}
	return nil
}

func (k Keeper) getVsAccountBudget(ctx context.Context, vsID uint64) (types.VsAccountBudget, bool, error) {
	budget, err := k.vsBudgetRef.S.Get(ctx, vsID)
	if errors.Is(err, collections.ErrNotFound) {
		return types.VsAccountBudget{}, false, nil
	}
	if err != nil {
		return types.VsAccountBudget{}, false, err
	}
	return budget, true, nil
}
