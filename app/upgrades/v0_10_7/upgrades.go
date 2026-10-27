package v0_10_7

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/verana-labs/verana-node/app/upgrades/types"
)

// BlockMaxGas replaces the unlimited block gas of the genesis consensus params.
const BlockMaxGas int64 = 100_000_000

// CreateUpgradeHandler bounds block gas and runs the module migrations that
// fill the pp and de lookup indexes from existing state.
func CreateUpgradeHandler(mm *module.Manager, configurator module.Configurator, baseApp types.BaseAppParamManager, _ types.AppKeepers) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		if err := setBlockMaxGas(sdk.UnwrapSDKContext(ctx), baseApp); err != nil {
			return nil, err
		}
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

func setBlockMaxGas(ctx sdk.Context, baseApp types.BaseAppParamManager) error {
	cp := baseApp.GetConsensusParams(ctx)
	if cp.Block == nil {
		return nil
	}
	cp.Block.MaxGas = BlockMaxGas
	return baseApp.StoreConsensusParams(ctx, cp)
}
