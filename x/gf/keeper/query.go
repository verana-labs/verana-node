package keeper

import "github.com/verana-labs/verana-node/x/gf/types"

type querier struct {
	Keeper
}

func NewQueryServerImpl(k Keeper) types.QueryServer {
	return &querier{Keeper: k}
}

var _ types.QueryServer = querier{}
