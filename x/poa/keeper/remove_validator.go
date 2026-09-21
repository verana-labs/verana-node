package keeper

import (
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/verana-labs/verana-node/x/poa/types"
)

func (k Keeper) removeValidator(ctx sdk.Context, validatorAddress string) error {
	accAddr, err := k.addressCodec.StringToBytes(validatorAddress)
	if err != nil {
		return err
	}
	valAddr := sdk.ValAddress(accAddr)

	validator, err := k.staking.GetValidator(ctx, valAddr)
	if err != nil {
		return errorsmod.Wrap(types.ErrNotAValidator, validatorAddress)
	}
	params, err := k.staking.GetParams(ctx)
	if err != nil {
		return err
	}
	if validator.IsBonded() {
		others, err := k.bondedValidators(ctx)
		if err != nil {
			return err
		}
		if others <= 1 {
			return types.ErrLastValidator
		}
	}

	if err := k.staking.Hooks().BeforeValidatorModified(ctx, valAddr); err != nil {
		return err
	}
	if validator.Tokens.IsPositive() {
		if err := k.staking.Hooks().BeforeValidatorSlashed(ctx, valAddr, math.LegacyOneDec()); err != nil {
			return err
		}
	}
	tokens := validator.Tokens
	changed, err := k.staking.RemoveValidatorTokens(ctx, validator, tokens)
	if err != nil {
		return err
	}
	coins := sdk.NewCoins(sdk.NewCoin(params.BondDenom, tokens))
	switch changed.GetStatus() {
	case stakingtypes.Bonded:
		err = k.bank.BurnCoins(ctx, stakingtypes.BondedPoolName, coins)
	case stakingtypes.Unbonding, stakingtypes.Unbonded:
		err = k.bank.BurnCoins(ctx, stakingtypes.NotBondedPoolName, coins)
	default:
		return types.ErrInvalidValidatorStatus
	}
	if err != nil {
		return err
	}
	if _, err := k.staking.Unbond(ctx, accAddr, valAddr, changed.DelegatorShares); err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeRemoveValidator,
		sdk.NewAttribute(types.AttributeKeyValidator, validatorAddress),
		sdk.NewAttribute(types.AttributeKeyTokens, tokens.String()),
	))
	return nil
}

func (k Keeper) bondedValidators(ctx sdk.Context) (int, error) {
	validators, err := k.staking.GetAllValidators(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, v := range validators {
		if v.IsBonded() && v.Tokens.IsPositive() {
			n++
		}
	}
	return n, nil
}
