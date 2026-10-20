package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/pp/types"
)

type Migrator struct {
	keeper Keeper
}

func NewMigrator(keeper Keeper) Migrator {
	return Migrator{keeper: keeper}
}

// Migrate1to2 fills the root and unrepaid-slash indexes from existing entries.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	return m.keeper.Participant.Walk(ctx, nil, func(_ uint64, p types.Participant) (bool, error) {
		return false, m.keeper.indexParticipantLookups(ctx, p)
	})
}
