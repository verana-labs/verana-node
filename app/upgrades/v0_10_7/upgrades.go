package v0_10_7

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/verana-labs/verana-node/app/upgrades/types"
)

// CreateUpgradeHandler runs the module migrations that fill the pp and de
// lookup indexes from existing state.
func CreateUpgradeHandler(mm *module.Manager, configurator module.Configurator, _ types.BaseAppParamManager, _ types.AppKeepers) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
