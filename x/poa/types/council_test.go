package types_test

import (
	"encoding/binary"
	"os"
	"strconv"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	groupkeeper "github.com/cosmos/cosmos-sdk/x/group/keeper"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/poa/types"
)

func TestCouncilPolicyAddress_MatchesGroupDerivation(t *testing.T) {
	dk := make([]byte, 8)
	binary.BigEndian.PutUint64(dk, 1)
	cred, err := authtypes.NewModuleCredential(group.ModuleName, []byte{groupkeeper.GroupPolicyTablePrefix}, dk)
	require.NoError(t, err)
	require.Equal(t, sdk.AccAddress(cred.Address()), types.CouncilPolicyAddress(1))
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

func TestCouncilGenesis_IsSelfAdministered(t *testing.T) {
	policy := types.CouncilAuthorityBech32("verana")
	members := []string{"verana16mzeyu9l6kua2cdg9x0jk5g6e7h0kk8q6uadu4", "verana1elf8m94agzfmpg2lkvqd776ellz7fxtgqnhcaa"}
	gs, err := types.CouncilGenesis(policy, members, "council", types.CouncilPercentage, time.Hour, time.Minute, time.Unix(0, 0).UTC())
	require.NoError(t, err)
	require.NoError(t, gs.Validate())
	require.Equal(t, policy, gs.Groups[0].Admin)
	require.Equal(t, policy, gs.GroupPolicies[0].Admin)
	require.Equal(t, policy, gs.GroupPolicies[0].Address)
	require.Equal(t, "2", gs.Groups[0].TotalWeight)
	require.Len(t, gs.GroupMembers, 2)
	require.EqualValues(t, 1, gs.GroupPolicySeq)
}

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("verana", "veranapub")
	cfg.SetBech32PrefixForValidator("veranavaloper", "veranavaloperpub")
	os.Exit(m.Run())
}
