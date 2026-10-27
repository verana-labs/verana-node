package v0_10_7

import (
	store "github.com/cosmos/cosmos-sdk/store/v2/types"

	"github.com/verana-labs/verana-node/app/upgrades/types"
)

const UpgradeName = "v0.10.7"

var Upgrade = types.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        store.StoreUpgrades{},
}
