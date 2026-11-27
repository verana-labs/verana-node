package keeper

import (
	"context"
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/gf/types"
)

func (q querier) Params(goCtx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	ngf, ng, err := q.networkVersionInForce(ctx)
	if err != nil && !errors.Is(err, types.ErrSubjectNotFound) {
		return nil, err
	}
	return &types.QueryParamsResponse{
		Params:                             q.GetParams(ctx),
		CouncilAccount:                     q.GetAuthority(),
		NetworkGovernanceFrameworkVersions: ng.NetworkGovernanceFrameworkVersions,
		NgfAcceptanceDeadline:              ng.NgfAcceptanceDeadline,
		NgfVersion:                         ngf.Version,
	}, nil
}
