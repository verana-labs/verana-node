package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/group"

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
