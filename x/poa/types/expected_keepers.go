package types

import (
	"context"
	"time"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/verana-labs/cosmos-group"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
)

type StakingKeeper interface {
	GetParams(ctx context.Context) (stakingtypes.Params, error)
	GetValidator(ctx context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error)
	GetAllValidators(ctx context.Context) ([]stakingtypes.Validator, error)
	GetAllDelegatorDelegations(ctx context.Context, delegator sdk.AccAddress) ([]stakingtypes.Delegation, error)
	GetDelegation(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress) (stakingtypes.Delegation, error)
	Undelegate(ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, shares math.LegacyDec) (time.Time, math.Int, error)
	ValidatorAddressCodec() address.Codec
}

// CorporationKeeper resolves a policy address through the AUTHZ-CHECK-5 index.
type CorporationKeeper interface {
	GetCorporationByPolicyAddress(ctx context.Context, policyAddress string) (cotypes.Corporation, error)
}

type GroupKeeper interface {
	GroupPolicyInfo(ctx context.Context, req *group.QueryGroupPolicyInfoRequest) (*group.QueryGroupPolicyInfoResponse, error)
	GroupMembers(ctx context.Context, req *group.QueryGroupMembersRequest) (*group.QueryGroupMembersResponse, error)
}

// MessageRouter is satisfied by *baseapp.MsgServiceRouter.
type MessageRouter interface {
	Handler(msg sdk.Msg) baseapp.MsgServiceHandler
}
