package upgrades_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/app/upgrades"
	v0_10_7 "github.com/verana-labs/verana-node/app/upgrades/v0_10_7"
)

func TestUpgradesRegistered(t *testing.T) {
	require.Len(t, upgrades.Upgrades, 1)
	require.Equal(t, v0_10_7.UpgradeName, upgrades.Upgrades[0].UpgradeName)
	require.Empty(t, upgrades.Upgrades[0].StoreUpgrades.Added)
	require.Empty(t, upgrades.Upgrades[0].StoreUpgrades.Deleted)
}
