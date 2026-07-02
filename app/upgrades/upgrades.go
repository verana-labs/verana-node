package upgrades

import (
	"github.com/verana-labs/verana-node/app/upgrades/types"
)

// Upgrades is the list of on-chain upgrade handlers registered by the app.
//
// It is intentionally empty. The upgrade machinery (see setupUpgradeHandlers /
// setupUpgradeStoreLoaders in app/app.go and the types package) is fully wired
// up, so adding a new upgrade only requires appending an entry to this slice.
//
// To add a new upgrade (e.g. "v1"):
//
//  1. Create a package under app/upgrades/v1/ with:
//     - constants.go: define `UpgradeName` and an exported `Upgrade`
//     (types.Upgrade) that wires in CreateUpgradeHandler and any
//     StoreUpgrades (added/deleted store keys). See the v9 example in git
//     history for the pattern.
//     - upgrades.go: define `CreateUpgradeHandler(...)` returning the
//     upgradetypes.UpgradeHandler, typically ending in mm.RunMigrations.
//
//  2. Import that package here and append its `Upgrade` to the slice below:
//
//     import v1 "github.com/verana-labs/verana-node/app/upgrades/v1"
//
//     var Upgrades = []types.Upgrade{
//     v1.Upgrade,
//     }
var Upgrades = []types.Upgrade{}
