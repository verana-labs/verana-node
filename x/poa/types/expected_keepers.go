package types

import (
	"context"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type StakingKeeper interface {
	GetParams(ctx context.Context) (stakingtypes.Params, error)
	GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
	GetAllValidators(ctx context.Context) ([]stakingtypes.Validator, error)
	GetAllDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress) ([]stakingtypes.Delegation, error)
	RemoveValidatorTokens(ctx context.Context, validator stakingtypes.Validator, tokensToRemove math.Int) (stakingtypes.Validator, error)
	Unbond(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, shares math.LegacyDec) (math.Int, error)
	Hooks() stakingtypes.StakingHooks
	ValidatorAddressCodec() address.Codec
}

type BankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

type GroupKeeper interface {
	GroupPolicyInfo(ctx context.Context, req *group.QueryGroupPolicyInfoRequest) (*group.QueryGroupPolicyInfoResponse, error)
	GroupMembers(ctx context.Context, req *group.QueryGroupMembersRequest) (*group.QueryGroupMembersResponse, error)
}

// MessageRouter is satisfied by *baseapp.MsgServiceRouter.
type MessageRouter interface {
	Handler(msg sdk.Msg) baseapp.MsgServiceHandler
}
