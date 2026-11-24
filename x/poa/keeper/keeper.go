package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/verana-labs/cosmos-group"

	"github.com/verana-labs/verana-node/x/poa/types"
)

type Keeper struct {
	authority    string
	addressCodec address.Codec
	staking      types.StakingKeeper
	bank         types.BankKeeper
	groups       types.GroupKeeper
	router       types.MessageRouter
}

func NewKeeper(authority string, addressCodec address.Codec, sk types.StakingKeeper, bk types.BankKeeper, gk types.GroupKeeper, router types.MessageRouter) Keeper {
	if _, err := addressCodec.StringToBytes(authority); err != nil {
		panic(fmt.Sprintf("invalid poa authority %q: %v", authority, err))
	}
	return Keeper{authority: authority, addressCodec: addressCodec, staking: sk, bank: bk, groups: gk, router: router}
}

func (k Keeper) GetAuthority() string { return k.authority }

func (k Keeper) CouncilGroupID(ctx context.Context) (uint64, error) {
	res, err := k.groups.GroupPolicyInfo(ctx, &group.QueryGroupPolicyInfoRequest{Address: k.authority})
	if err != nil {
		return 0, fmt.Errorf("no council group policy at %s: %w", k.authority, err)
	}
	return res.Info.GroupId, nil
}

func (k Keeper) IsCouncilMember(ctx context.Context, addr string) (bool, error) {
	members, err := k.CouncilMembers(ctx)
	if err != nil {
		return false, err
	}
	for _, m := range members {
		if m == addr {
			return true, nil
		}
	}
	return false, nil
}

func (k Keeper) CouncilMembers(ctx context.Context) ([]string, error) {
	groupID, err := k.CouncilGroupID(ctx)
	if err != nil {
		return nil, err
	}
	res, err := k.groups.GroupMembers(ctx, &group.QueryGroupMembersRequest{GroupId: groupID, Pagination: &query.PageRequest{Limit: 1000}})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(res.Members))
	for _, m := range res.Members {
		out = append(out, m.Member.Address)
	}
	return out, nil
}

func (k Keeper) ValidatorBond(ctx context.Context) (sdk.Coin, error) {
	params, err := k.staking.GetParams(ctx)
	if err != nil {
		return sdk.Coin{}, err
	}
	return types.ValidatorBond(params.BondDenom), nil
}

// seatedValidators counts records that still hold the bond; removed ones linger at zero until unbonding_time.
func (k Keeper) seatedValidators(ctx context.Context) (int, error) {
	validators, err := k.staking.GetAllValidators(ctx)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, v := range validators {
		if v.Tokens.IsPositive() {
			n++
		}
	}
	return n, nil
}

func (k Keeper) hasActiveValidator(ctx context.Context, addr string) (bool, error) {
	acc, err := k.addressCodec.StringToBytes(addr)
	if err != nil {
		return false, err
	}
	v, err := k.staking.GetValidator(ctx, sdk.ValAddress(acc))
	if errors.Is(err, stakingtypes.ErrNoValidatorFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v.Tokens.IsPositive(), nil
}

func (k Keeper) councilPolicy(ctx context.Context) (group.DecisionPolicy, error) {
	res, err := k.groups.GroupPolicyInfo(ctx, &group.QueryGroupPolicyInfoRequest{Address: k.authority})
	if err != nil {
		return nil, fmt.Errorf("no council group policy at %s: %w", k.authority, err)
	}
	return res.Info.GetDecisionPolicy()
}
