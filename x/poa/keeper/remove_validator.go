package keeper

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/poa/types"
)

// removeValidator undelegates the self-bond; it returns to the operator after unbonding_time.
func (k Keeper) removeValidator(ctx sdk.Context, validatorAddress string) error {
	accAddr, err := k.addressCodec.StringToBytes(validatorAddress)
	if err != nil {
		return err
	}
	valAddr := sdk.ValAddress(accAddr)

	validator, err := k.staking.GetValidator(ctx, valAddr)
	if err != nil || !validator.Tokens.IsPositive() {
		return errorsmod.Wrap(types.ErrNotAValidator, validatorAddress)
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

	delegation, err := k.staking.GetDelegation(ctx, accAddr, valAddr)
	if err != nil {
		return errorsmod.Wrap(types.ErrNotAValidator, err.Error())
	}
	_, amount, err := k.staking.Undelegate(ctx, accAddr, valAddr, delegation.Shares)
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeRemoveValidator,
		sdk.NewAttribute(types.AttributeKeyValidator, validatorAddress),
		sdk.NewAttribute(types.AttributeKeyTokens, amount.String()),
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
