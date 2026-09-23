package keeper

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/collections"
	feegrant "cosmossdk.io/x/feegrant"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/de/types"
)

// GrantFeeAllowance implements [MOD-DE-MSG-1] with a fresh budget. The MSG-5-5
// recompute uses grantFeeAllowance to carry the spent budget instead.
func (k Keeper) GrantFeeAllowance(
	goCtx context.Context,
	grantorCorporationID uint64,
	grantee string,
	msgTypes []string,
	expiration *time.Time,
	spendLimit sdk.Coins,
	period *time.Duration,
) error {
	return k.grantFeeAllowance(goCtx, grantorCorporationID, grantee, msgTypes, expiration, spendLimit, period, spendLimit)
}

// grantFeeAllowance is GrantFeeAllowance with an explicit remaining budget.
func (k Keeper) grantFeeAllowance(
	goCtx context.Context,
	grantorCorporationID uint64,
	grantee string,
	msgTypes []string,
	expiration *time.Time,
	spendLimit sdk.Coins,
	period *time.Duration,
	remaining sdk.Coins,
) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DE-MSG-1-2] Basic checks.

	// msg_types MUST be a list of VPR delegable messages only.
	if len(msgTypes) == 0 {
		return fmt.Errorf("msg_types must not be empty")
	}
	for _, mt := range msgTypes {
		if !types.VPRDelegableMsgTypes[mt] {
			return fmt.Errorf("%w: %s", types.ErrInvalidMsgType, mt)
		}
	}

	// expiration if specified MUST be in the future.
	if expiration != nil && !expiration.After(now) {
		return types.ErrExpirationInPast
	}

	// spend_limit if specified MUST be valid.
	if len(spendLimit) > 0 && !spendLimit.IsValid() {
		return types.ErrInvalidSpendLimit
	}

	// period if specified MUST be valid (positive).
	if period != nil && *period <= 0 {
		return fmt.Errorf("period must be a positive duration")
	}

	// if period is specified, expiration MUST also be specified [MOD-DE-MSG-1-2].
	if period != nil && expiration == nil {
		return fmt.Errorf("expiration must be specified when period is set")
	}

	// [MOD-DE-MSG-1-4] Execution.
	key := collections.Join(grantorCorporationID, grantee)
	feeGrant := types.FeeGrant{
		GrantorCorporationId: grantorCorporationID,
		Grantee:              grantee,
		MsgTypes:             msgTypes,
		SpendLimit:           spendLimit,
		RemainingSpend:       remaining,
		Expiration:           expiration,
		Period:               period,
	}
	if err := k.FeeGrants.Set(ctx, key, feeGrant); err != nil {
		return fmt.Errorf("failed to set FeeGrant: %w", err)
	}

	// Realize the on-chain x/feegrant allowance (MOD-DE-MSG-1-4).
	if fk := k.feegrantKeeper(); fk != nil {
		granter, granteeAddr, err := k.feeGrantAddrs(ctx, grantorCorporationID, grantee)
		if err != nil {
			return err
		}
		var inner feegrant.FeeAllowanceI
		if len(spendLimit) > 0 && period != nil {
			// [MOD-DE-MSG-1-4] No absolute expiration: auto-renews until revoked.
			inner = &feegrant.PeriodicAllowance{
				Period:           *period,
				PeriodSpendLimit: spendLimit,
				PeriodCanSpend:   remaining,
				PeriodReset:      *expiration,
			}
		} else {
			inner = &feegrant.BasicAllowance{SpendLimit: spendLimit, Expiration: expiration}
		}
		allowed, err := feegrant.NewAllowedMsgAllowance(inner, msgTypes)
		if err != nil {
			return fmt.Errorf("build fee allowance: %w", err)
		}
		// "create or update" => revoke-then-grant (x/feegrant rejects an existing grant).
		if _, gerr := fk.GetAllowance(ctx, granter, granteeAddr); gerr == nil {
			if err := fk.RevokeAllowance(ctx, granter, granteeAddr); err != nil {
				return fmt.Errorf("revoke existing allowance: %w", err)
			}
		}
		if err := fk.GrantAllowance(ctx, granter, granteeAddr, allowed); err != nil {
			return fmt.Errorf("grant fee allowance: %w", err)
		}
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeGrantFeeAllowance,
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(grantorCorporationID, 10)),
			sdk.NewAttribute(types.AttributeKeyGrantee, grantee),
			sdk.NewAttribute(types.AttributeKeyTimestamp, now.String()),
		),
	)
	return nil
}

// RevokeFeeAllowance implements [MOD-DE-MSG-2]. It removes the FeeGrant for the
// composite (grantor_corporation_id, grantee). This is an internal method called
// by GrantOperatorAuthorization, RevokeOperatorAuthorization, and the MSG-5-5
// recompute subroutine.
func (k Keeper) RevokeFeeAllowance(goCtx context.Context, grantorCorporationID uint64, grantee string) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-DE-MSG-2-2] Basic checks.
	if grantorCorporationID == 0 {
		return fmt.Errorf("grantor_corporation_id must be specified")
	}
	if grantee == "" {
		return fmt.Errorf("grantee must be specified")
	}

	// Revoke the on-chain x/feegrant allowance if present (MOD-DE-MSG-2-4).
	if fk := k.feegrantKeeper(); fk != nil {
		granter, granteeAddr, err := k.feeGrantAddrs(ctx, grantorCorporationID, grantee)
		if err != nil {
			return err
		}
		if _, gerr := fk.GetAllowance(ctx, granter, granteeAddr); gerr == nil {
			if err := fk.RevokeAllowance(ctx, granter, granteeAddr); err != nil {
				return err
			}
		}
	}

	// [MOD-DE-MSG-2-4] Execution: if FeeGrant exists, delete it, else do nothing.
	key := collections.Join(grantorCorporationID, grantee)
	has, err := k.FeeGrants.Has(ctx, key)
	if err != nil {
		return err
	}
	if !has {
		return nil
	}
	if err := k.FeeGrants.Remove(ctx, key); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRevokeFeeAllowance,
			sdk.NewAttribute(types.AttributeKeyCorporationID, strconv.FormatUint(grantorCorporationID, 10)),
			sdk.NewAttribute(types.AttributeKeyGrantee, grantee),
			sdk.NewAttribute(types.AttributeKeyTimestamp, ctx.BlockTime().String()),
		),
	)
	return nil
}

// carriedFeeBudget keeps the current reset and subtracts what was already spent
// from the new total, floored at zero; fresh once the period has elapsed.
func (k Keeper) carriedFeeBudget(ctx context.Context, grantorCorporationID uint64, grantee string, total sdk.Coins, period time.Duration) (time.Time, sdk.Coins) {
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	fresh := now.Add(period)
	fk := k.feegrantKeeper()
	if fk == nil {
		return fresh, total
	}
	granter, granteeAddr, err := k.feeGrantAddrs(ctx, grantorCorporationID, grantee)
	if err != nil {
		return fresh, total
	}
	cur, err := fk.GetAllowance(ctx, granter, granteeAddr)
	if err != nil {
		return fresh, total
	}
	if wrapped, ok := cur.(*feegrant.AllowedMsgAllowance); ok {
		if cur, err = wrapped.GetAllowance(); err != nil {
			return fresh, total
		}
	}
	pa, ok := cur.(*feegrant.PeriodicAllowance)
	if !ok || !pa.PeriodReset.After(now) {
		return fresh, total
	}
	remaining := sdk.NewCoins()
	for _, c := range total {
		spent := pa.PeriodSpendLimit.AmountOf(c.Denom).Sub(pa.PeriodCanSpend.AmountOf(c.Denom))
		left := c.Amount.Sub(spent)
		if left.IsPositive() {
			remaining = remaining.Add(sdk.NewCoin(c.Denom, left))
		}
	}
	return pa.PeriodReset, remaining
}

// feeGrantAddrs resolves the granter (corp policy_address) and grantee accounts.
func (k Keeper) feeGrantAddrs(ctx context.Context, grantorCorporationID uint64, grantee string) (sdk.AccAddress, sdk.AccAddress, error) {
	co, err := k.corporationKeeper().ResolveCorporationByID(ctx, grantorCorporationID)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve corporation %d: %w", grantorCorporationID, err)
	}
	granter, err := sdk.AccAddressFromBech32(co.PolicyAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid corporation policy address: %w", err)
	}
	granteeAddr, err := sdk.AccAddressFromBech32(grantee)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid grantee address: %w", err)
	}
	return granter, granteeAddr, nil
}
