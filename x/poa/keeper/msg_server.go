package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/poa/types"
)

type msgServer struct{ Keeper }

func NewMsgServerImpl(k Keeper) types.MsgServer { return &msgServer{Keeper: k} }

var _ types.MsgServer = msgServer{}

func (m msgServer) AddValidator(goCtx context.Context, msg *types.MsgAddValidator) (*types.MsgAddValidatorResponse, error) {
	if msg.Authority != m.authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "expected %s got %s", m.authority, msg.Authority)
	}
	if err := m.addValidator(sdk.UnwrapSDKContext(goCtx), msg); err != nil {
		return nil, err
	}
	return &types.MsgAddValidatorResponse{}, nil
}

func (m msgServer) RemoveValidator(goCtx context.Context, msg *types.MsgRemoveValidator) (*types.MsgRemoveValidatorResponse, error) {
	if msg.Authority != m.authority {
		return nil, errorsmod.Wrapf(types.ErrInvalidSigner, "expected %s got %s", m.authority, msg.Authority)
	}
	if err := m.removeValidator(sdk.UnwrapSDKContext(goCtx), msg.ValidatorAddress); err != nil {
		return nil, err
	}
	return &types.MsgRemoveValidatorResponse{}, nil
}

func (m msgServer) SelfRemoveValidator(goCtx context.Context, msg *types.MsgSelfRemoveValidator) (*types.MsgSelfRemoveValidatorResponse, error) {
	if err := m.removeValidator(sdk.UnwrapSDKContext(goCtx), msg.ValidatorAddress); err != nil {
		return nil, err
	}
	return &types.MsgSelfRemoveValidatorResponse{}, nil
}
