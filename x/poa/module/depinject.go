package poa

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"
	"github.com/cosmos/cosmos-sdk/baseapp"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	groupkeeper "github.com/verana-labs/cosmos-group/keeper"

	cokeeper "github.com/verana-labs/verana-node/x/co/keeper"
	"github.com/verana-labs/verana-node/x/poa/keeper"
	"github.com/verana-labs/verana-node/x/poa/types"
)

var _ depinject.OnePerModuleType = AppModule{}

func init() {
	appconfig.Register(
		&types.Module{},
		appconfig.Provide(ProvideModule),
	)
}

type ModuleInputs struct {
	depinject.In

	Config        *types.Module
	AddressCodec  address.Codec
	StakingKeeper *stakingkeeper.Keeper
	CoKeeper      cokeeper.Keeper
	GroupKeeper   groupkeeper.Keeper
	Router        *baseapp.MsgServiceRouter
}

type ModuleOutputs struct {
	depinject.Out

	PoaKeeper keeper.Keeper
	Module    appmodule.AppModule
}

func ProvideModule(in ModuleInputs) ModuleOutputs {
	if in.Config.Authority == "" {
		panic("x/poa requires an explicit authority")
	}
	k := keeper.NewKeeper(in.Config.Authority, in.AddressCodec, in.StakingKeeper, in.CoKeeper, in.GroupKeeper, in.Router)
	return ModuleOutputs{PoaKeeper: k, Module: NewAppModule(k)}
}
