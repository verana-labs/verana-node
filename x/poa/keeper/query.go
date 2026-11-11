package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/poa/types"
)

type queryServer struct{ Keeper }

func NewQueryServerImpl(k Keeper) types.QueryServer { return queryServer{Keeper: k} }

var _ types.QueryServer = queryServer{}

func (q queryServer) Council(ctx context.Context, _ *types.QueryCouncilRequest) (*types.QueryCouncilResponse, error) {
	groupID, err := q.CouncilGroupID(ctx)
	if err != nil {
		return nil, err
	}
	members, err := q.CouncilMembers(ctx)
	if err != nil {
		return nil, err
	}
	seated := map[string]bool{}
	for _, m := range members {
		seated[m] = true
	}
	var unseated []string
	validators, err := q.staking.GetAllValidators(ctx)
	if err != nil {
		return nil, err
	}
	for _, v := range validators {
		if !v.Tokens.IsPositive() {
			continue
		}
		valBz, err := q.staking.ValidatorAddressCodec().StringToBytes(v.OperatorAddress)
		if err != nil {
			return nil, err
		}
		operator, err := q.addressCodec.BytesToString(valBz)
		if err != nil {
			return nil, err
		}
		if !seated[operator] {
			unseated = append(unseated, operator)
		}
	}
	var missing []string
	for _, m := range members {
		acc, err := q.addressCodec.StringToBytes(m)
		if err != nil {
			return nil, err
		}
		v, err := q.staking.GetValidator(ctx, sdk.ValAddress(acc))
		if err != nil || !v.IsBonded() {
			missing = append(missing, m)
		}
	}
	return &types.QueryCouncilResponse{Authority: q.authority, GroupId: groupID, Members: members, MembersWithoutValidator: missing, ValidatorsWithoutSeat: unseated}, nil
}
