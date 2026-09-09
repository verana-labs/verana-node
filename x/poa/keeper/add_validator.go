package keeper

import (
	"errors"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/verana-labs/verana-node/x/poa/types"
)

func (k Keeper) addValidator(ctx sdk.Context, msg *types.MsgAddValidator) error {
	accAddr, err := k.addressCodec.StringToBytes(msg.ValidatorAddress)
	if err != nil {
		return err
	}
	valAddr := sdk.ValAddress(accAddr)

	member, err := k.IsCouncilMember(ctx, msg.ValidatorAddress)
	if err != nil {
		return err
	}
	if !member {
		return errorsmod.Wrap(types.ErrNotCouncilMember, msg.ValidatorAddress)
	}

	params, err := k.staking.GetParams(ctx)
	if err != nil {
		return err
	}
	validators, err := k.staking.GetAllValidators(ctx)
	if err != nil {
		return err
	}
	if uint32(len(validators)) >= params.MaxValidators {
		return types.ErrMaxValidatorsReached
	}

	validator, err := k.staking.GetValidator(ctx, valAddr)
	switch {
	case err == nil && !validator.Tokens.IsZero():
		return types.ErrAddressHasBondedTokens
	case err != nil && !errors.Is(err, stakingtypes.ErrNoValidatorFound):
		return err
	}

	delegations, err := k.staking.GetAllDelegatorDelegations(ctx, accAddr)
	if err != nil {
		return err
	}
	for _, d := range delegations {
		if d.Shares.IsPositive() {
			return types.ErrAddressHasDelegations
		}
	}
	bond := types.ValidatorBond(params.BondDenom)
	if msg.Pubkey == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidType, "pubkey is required")
	}
	pubKey, ok := msg.Pubkey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return errorsmod.Wrap(sdkerrors.ErrInvalidType, "pubkey must be a cryptotypes.PubKey")
	}
	valStr, err := k.staking.ValidatorAddressCodec().BytesToString(valAddr)
	if err != nil {
		return err
	}
	description := stakingtypes.NewDescription(msg.Description.Moniker, msg.Description.Identity, msg.Description.Website, msg.Description.SecurityContact, msg.Description.Details)
	createMsg, err := stakingtypes.NewMsgCreateValidator(
		valStr, pubKey, bond, description,
		stakingtypes.NewCommissionRates(sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec(), sdkmath.LegacyZeroDec()),
		sdkmath.OneInt(),
	)
	if err != nil {
		return err
	}
	if err := createMsg.Validate(k.staking.ValidatorAddressCodec()); err != nil {
		return err
	}
	handler := k.router.Handler(createMsg)
	if handler == nil {
		return errorsmod.Wrap(sdkerrors.ErrUnknownRequest, "no handler for MsgCreateValidator")
	}

	coins := sdk.NewCoins(bond)
	if err := k.bank.MintCoins(ctx, types.ModuleName, coins); err != nil {
		return err
	}
	if err := k.bank.SendCoinsFromModuleToAccount(ctx, types.ModuleName, accAddr, coins); err != nil {
		return err
	}
	res, err := handler(ctx, createMsg)
	if err != nil {
		return err
	}
	ctx.EventManager().EmitEvents(res.GetEvents())

	ctx.EventManager().EmitEvent(sdk.NewEvent(types.EventTypeAddValidator,
		sdk.NewAttribute(types.AttributeKeyValidator, msg.ValidatorAddress),
		sdk.NewAttribute(types.AttributeKeyTokens, bond.Amount.String()),
	))
	return nil
}
