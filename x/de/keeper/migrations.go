package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/de/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 fills the vs_operator index from existing authorizations.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return m.keeper.VSOperatorAuthorizations.Walk(ctx, nil, func(_ uint64, vsoa types.VSOperatorAuthorization) (bool, error) {
		return false, m.keeper.VSOACorpByOperator.Set(ctx, vsoa.VsOperator, vsoa.CorporationId)
	})
}
