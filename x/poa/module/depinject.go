package poa

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/appmodule"
	"cosmossdk.io/depinject"
	"cosmossdk.io/depinject/appconfig"
	"github.com/cosmos/cosmos-sdk/baseapp"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

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
	BankKeeper    bankkeeper.Keeper
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
	k := keeper.NewKeeper(in.Config.Authority, in.AddressCodec, in.StakingKeeper, in.BankKeeper, in.GroupKeeper, in.Router)
	return ModuleOutputs{PoaKeeper: k, Module: NewAppModule(k)}
}
