package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/comet"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/verana-labs/verana-node/util/events"
	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/ds/types"
	tdtypes "github.com/verana-labs/verana-node/x/td/types"
)

// CountBlock adds the block to the open epoch: a member block for every
// validator of the set, jailed ones included, and a signed block for each
// whose signature the block's last commit carries.
func (k Keeper) CountBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	signed := make(map[string]bool, len(sdkCtx.VoteInfos()))
	for _, v := range sdkCtx.VoteInfos() {
		signed[string(v.Validator.Address)] = comet.BlockIDFlag(v.BlockIdFlag) != comet.BlockIDFlagAbsent
	}

	validators, err := k.staking.GetAllValidators(ctx)
	if err != nil {
		return err
	}
	for _, v := range validators {
		// removed validators keep a record with no tokens until unbonding completes
		if !v.Tokens.IsPositive() {
			continue
		}
		op, err := k.staking.ValidatorAddressCodec().StringToBytes(v.OperatorAddress)
		if err != nil {
			return err
		}
		cons, err := v.GetConsAddr()
		if err != nil {
			return err
		}
		b, err := k.ValidatorBlocks.Get(ctx, op)
		if err != nil && !errors.Is(err, collections.ErrNotFound) {
			return err
		}
		b.MemberBlocks++
		if signed[string(cons)] {
			b.SignedBlocks++
		}
		if err := k.ValidatorBlocks.Set(ctx, op, b); err != nil {
			return err
		}
	}

	n, err := k.epochBlocks(ctx)
	if err != nil {
		return err
	}
	return k.EpochBlocks.Set(ctx, n+1)
}

// PayoutEpoch performs steps (1) and (2) of the boundary of epoch: it pays out
// the pool per [DIST-1] from the counters of the closed epoch, or skips the
// payout when the epoch has no block or no valid rate, then resets the counters.
func (k Keeper) PayoutEpoch(ctx context.Context, epoch uint64) error {
	epochBlocks, err := k.epochBlocks(ctx)
	if err != nil {
		return err
	}
	var addrs []sdk.ValAddress
	var blocks []types.ValidatorBlocks
	err = k.ValidatorBlocks.Walk(ctx, nil, func(addr sdk.ValAddress, b types.ValidatorBlocks) (bool, error) {
		addrs, blocks = append(addrs, addr), append(blocks, b)
		return false, nil
	})
	if err != nil {
		return err
	}
	if err := k.ValidatorBlocks.Clear(ctx, nil); err != nil {
		return err
	}
	if err := k.EpochBlocks.Remove(ctx); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	ev := events.New(types.ModuleName, types.EventTypeCloseEpoch).
		Uint(types.AttributeKeyEpoch, epoch).
		Uint(types.AttributeKeyEpochBlocks, epochBlocks)
	if epochBlocks == 0 {
		ev.Bool(types.AttributeKeySkipped, true).Emit(sdkCtx)
		return nil
	}
	if _, _, err := k.xr.GenesisRate(ctx); err != nil {
		ev.Bool(types.AttributeKeySkipped, true).Emit(sdkCtx)
		return nil
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}
	denom := tdtypes.BondDenom
	fiat := k.td.K.MainFiatCurrency(ctx)
	var budgets [3]math.Int
	for i, b := range []math.LegacyDec{params.ValidatorNodeBudget, params.CouncilBudget, params.FoundationBudget} {
		budgets[i], err = k.toNative(ctx, fiat, denom, b)
		if err != nil {
			ev.Bool(types.AttributeKeySkipped, true).Emit(sdkCtx)
			return nil
		}
	}
	nodeBudget, councilBudget, foundationBudget := budgets[0], budgets[1], budgets[2]
	ev.Bool(types.AttributeKeySkipped, false)

	pool := k.bank.GetBalance(ctx, authtypes.NewModuleAddress(types.ModuleName), denom).Amount
	dues := make([]math.Int, len(blocks))
	total := math.ZeroInt()
	for i, b := range blocks {
		dues[i] = validatorPayout(nodeBudget, b, epochBlocks, params)
		total = total.Add(dues[i])
	}
	paid := math.ZeroInt()
	for i, addr := range addrs {
		// a pool short of the validators' total is shared pro rata
		if total.GT(pool) {
			dues[i] = dues[i].Mul(pool).Quo(total)
		}
		if err := k.pay(ctx, sdk.AccAddress(addr), denom, dues[i]); err != nil {
			return err
		}
		ev.Addr(types.AttributeKeyValidator, sdk.AccAddress(addr)).
			Uint(types.AttributeKeySignedBlocks, blocks[i].SignedBlocks).
			Uint(types.AttributeKeyMemberBlocks, blocks[i].MemberBlocks).
			Int(types.AttributeKeyValidatorPayout, dues[i])
		paid = paid.Add(dues[i])
	}
	pool = pool.Sub(paid)

	council := math.MinInt(councilBudget, pool)
	if err := k.pay(ctx, k.authority, denom, council); err != nil {
		return err
	}
	pool = pool.Sub(council)
	foundation := math.MinInt(foundationBudget, pool)
	if err := k.pay(ctx, sdk.MustAccAddressFromBech32(params.FoundationAccount), denom, foundation); err != nil {
		return err
	}
	pool = pool.Sub(foundation)

	// no bonding mechanism exists yet, so item 4 pays nothing whatever bonding_enabled says
	if pool.IsPositive() {
		if err := k.bank.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(sdk.NewCoin(denom, pool))); err != nil {
			return err
		}
	}
	ev.Int(types.AttributeKeyCouncilPayout, council).
		Int(types.AttributeKeyFoundationPayout, foundation).
		Int(types.AttributeKeyBurned, pool).
		Emit(sdkCtx)
	return nil
}

// validatorPayout is the budget pro-rated to the validator's share of the
// epoch, docked by its signed-block ratio.
func validatorPayout(budget math.Int, b types.ValidatorBlocks, epochBlocks uint64, p types.Params) math.Int {
	ratio := math.LegacyNewDecFromInt(math.NewIntFromUint64(b.SignedBlocks)).QuoTruncate(math.LegacyNewDecFromInt(math.NewIntFromUint64(b.MemberBlocks)))
	var factor math.LegacyDec
	switch {
	case ratio.GTE(p.ValidatorTargetUptime):
		factor = math.LegacyOneDec()
	case ratio.LT(p.ValidatorMinUptime):
		return math.ZeroInt()
	default:
		factor = ratio.Sub(p.ValidatorMinUptime).QuoTruncate(p.ValidatorTargetUptime.Sub(p.ValidatorMinUptime))
	}
	return math.LegacyNewDecFromInt(budget.Mul(math.NewIntFromUint64(b.MemberBlocks))).
		QuoTruncate(math.LegacyNewDecFromInt(math.NewIntFromUint64(epochBlocks))).
		MulTruncate(factor).
		TruncateInt()
}

// toNative converts a budget in minor units of the main fiat currency through Get Price.
func (k Keeper) toNative(ctx context.Context, fiat, denom string, amount math.LegacyDec) (math.Int, error) {
	return k.xr.GetPrice(ctx, cstypes.PricingAssetType_FIAT, fiat, cstypes.PricingAssetType_COIN, denom, amount.TruncateInt())
}

func (k Keeper) pay(ctx context.Context, to sdk.AccAddress, denom string, amount math.Int) error {
	if !amount.IsPositive() {
		return nil
	}
	return k.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, to, sdk.NewCoins(sdk.NewCoin(denom, amount)))
}

func (k Keeper) epochBlocks(ctx context.Context) (uint64, error) {
	n, err := k.EpochBlocks.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return 0, nil
	}
	return n, err
}
