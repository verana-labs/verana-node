package keeper

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant"

	"github.com/verana-labs/verana-node/util/events"
	"github.com/verana-labs/verana-node/x/de/types"
)

// GrantFeeAllowance implements [MOD-DE-MSG-1].
func (k Keeper) GrantFeeAllowance(
	goCtx context.Context,
	grantorCorporationID uint64,
	grantee string,
	msgTypes []types.MsgType,
	expiration *time.Time,
	spendLimit sdk.Coins,
	period *time.Duration,
) error {
	ctx := sdk.UnwrapSDKContext(goCtx)
	now := ctx.BlockTime()

	// [MOD-DE-MSG-1-2] Basic checks.

	// msg_types MUST be delegable values, plus AcceptCorporationGovernanceFramework.
	if err := types.ValidateMsgTypes(msgTypes, types.FeeGrantMsgTypes); err != nil {
		return err
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

	key := collections.Join(grantorCorporationID, grantee)
	existing, err := k.getFeeGrant(ctx, key)
	if err != nil {
		return err
	}
	fk := k.feegrantKeeper()
	var granter, granteeAddr sdk.AccAddress
	var current feegrant.FeeAllowanceI
	if fk != nil {
		if granter, granteeAddr, err = k.feeGrantAddrs(ctx, grantorCorporationID, grantee); err != nil {
			return err
		}
		// The module never overwrites an allowance it did not create.
		if current, _ = fk.GetAllowance(ctx, granter, granteeAddr); current != nil {
			owned, err := ownsAllowance(existing, current, now)
			if err != nil {
				return err
			}
			if !owned {
				return types.ErrFeeAllowanceNotOwned
			}
		}
	}

	// [MOD-DE-MSG-1-4] Execution.
	feeGrant := types.FeeGrant{
		GrantorCorporationId: grantorCorporationID,
		Grantee:              grantee,
		MsgTypes:             msgTypes,
		SpendLimit:           spendLimit,
		RemainingSpend:       spendLimit,
		Expiration:           expiration,
	}
	if len(spendLimit) > 0 {
		feeGrant.Period = period
	}
	if err := k.FeeGrants.Set(ctx, key, feeGrant); err != nil {
		return fmt.Errorf("failed to set FeeGrant: %w", err)
	}

	if fk != nil {
		var inner feegrant.FeeAllowanceI
		if feeGrant.Period != nil {
			// No absolute expiration: auto-renews until revoked.
			inner = &feegrant.PeriodicAllowance{
				Period:           *period,
				PeriodSpendLimit: spendLimit,
				PeriodCanSpend:   spendLimit,
				PeriodReset:      *expiration,
			}
		} else {
			inner = &feegrant.BasicAllowance{SpendLimit: spendLimit, Expiration: expiration}
		}
		urls, err := types.MsgTypeURLs(msgTypes)
		if err != nil {
			return err
		}
		allowed, err := feegrant.NewAllowedMsgAllowance(inner, urls)
		if err != nil {
			return fmt.Errorf("build fee allowance: %w", err)
		}
		// x/feegrant rejects a grant over an existing one.
		if current != nil {
			if err := fk.RevokeAllowance(ctx, granter, granteeAddr); err != nil {
				return fmt.Errorf("revoke existing allowance: %w", err)
			}
		}
		if err := fk.GrantAllowance(ctx, granter, granteeAddr, allowed); err != nil {
			return fmt.Errorf("grant fee allowance: %w", err)
		}
	}

	feeAllowanceEvent(types.EventTypeGrantFeeAllowance, grantorCorporationID, grantee).Emit(ctx)
	return nil
}

// RevokeFeeAllowance implements [MOD-DE-MSG-2].
func (k Keeper) RevokeFeeAllowance(goCtx context.Context, grantorCorporationID uint64, grantee string) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// [MOD-DE-MSG-2-2] Basic checks.
	if grantorCorporationID == 0 {
		return fmt.Errorf("grantor_corporation_id must be specified")
	}
	if grantee == "" {
		return fmt.Errorf("grantee must be specified")
	}

	// [MOD-DE-MSG-2-4] Execution: with no FeeGrant, nothing is touched.
	key := collections.Join(grantorCorporationID, grantee)
	existing, err := k.getFeeGrant(ctx, key)
	if err != nil {
		return err
	}
	if existing == nil {
		return nil
	}
	if err := k.FeeGrants.Remove(ctx, key); err != nil {
		return err
	}

	if fk := k.feegrantKeeper(); fk != nil {
		granter, granteeAddr, err := k.feeGrantAddrs(ctx, grantorCorporationID, grantee)
		if err != nil {
			return err
		}
		if current, _ := fk.GetAllowance(ctx, granter, granteeAddr); current != nil {
			owned, err := ownsAllowance(existing, current, ctx.BlockTime())
			if err != nil {
				return err
			}
			if owned {
				if err := fk.RevokeAllowance(ctx, granter, granteeAddr); err != nil {
					return err
				}
			}
		}
	}

	feeAllowanceEvent(types.EventTypeRevokeFeeAllowance, grantorCorporationID, grantee).Emit(ctx)
	return nil
}

func (k Keeper) getFeeGrant(ctx context.Context, key collections.Pair[uint64, string]) (*types.FeeGrant, error) {
	fg, err := k.FeeGrants.Get(ctx, key)
	if errors.Is(err, collections.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &fg, nil
}

// ownsAllowance is false with no entry, or when the entry ended by expiry while the allowance has not.
func ownsAllowance(fg *types.FeeGrant, allowance feegrant.FeeAllowanceI, now time.Time) (bool, error) {
	if fg == nil {
		return false, nil
	}
	if fg.Period != nil || fg.Expiration == nil || fg.Expiration.After(now) {
		return true, nil
	}
	exp, err := allowance.ExpiresAt()
	if err != nil {
		return false, err
	}
	return exp != nil && !exp.After(now), nil
}

func feeAllowanceEvent(eventType string, grantorCorporationID uint64, grantee string) *events.Event {
	return events.New(types.ModuleName, eventType).
		Uint(types.AttributeKeyGrantorCorporationID, grantorCorporationID).
		Str(types.AttributeKeyGrantee, grantee)
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
