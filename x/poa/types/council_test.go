package types_test

import (
	"encoding/binary"
	"os"
	"strconv"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"github.com/verana-labs/cosmos-group"
	groupkeeper "github.com/verana-labs/cosmos-group/keeper"

	"github.com/verana-labs/verana-node/x/poa/types"
)

func TestGroupPolicyAddress_MatchesGroupDerivation(t *testing.T) {
	dk := make([]byte, 8)
	binary.BigEndian.PutUint64(dk, 2)
	cred, err := authtypes.NewModuleCredential(group.ModuleName, []byte{groupkeeper.GroupPolicyTablePrefix}, dk)
	require.NoError(t, err)
	require.Equal(t, sdk.AccAddress(cred.Address()), types.GroupPolicyAddress(2))
}

func TestCouncilAuthorityBech32_Pinned(t *testing.T) {
	require.Equal(t,
		"verana1afk9zr2hn2jsac63h4hm60vl9z3e5u69gndzf7c99cqge3vzwjzsh3z8fv",
		types.CouncilAuthorityBech32("verana"))
}

func TestCouncilPercentage_MatchesCeilTwoThirds(t *testing.T) {
	policy := group.PercentageDecisionPolicy{Percentage: types.CouncilPercentage}
	for n := 1; n <= 25; n++ {
		need := (2*n + 2) / 3
		for yes := 0; yes <= n; yes++ {
			tally := group.TallyResult{
				YesCount: strconv.Itoa(yes), NoCount: strconv.Itoa(n - yes), AbstainCount: "0", NoWithVetoCount: "0",
			}
			res, err := policy.Allow(tally, strconv.Itoa(n))
			require.NoError(t, err)
			require.Equalf(t, yes >= need, res.Allow, "seats=%d yes=%d need=%d", n, yes, need)
		}
	}
}

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("verana", "veranapub")
	cfg.SetBech32PrefixForValidator("veranavaloper", "veranavaloperpub")
	os.Exit(m.Run())
}
