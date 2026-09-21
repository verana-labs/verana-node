package types

import (
	"encoding/binary"
	"strconv"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/group"
)

// groupPolicyTablePrefix mirrors x/group's keeper.GroupPolicyTablePrefix (pinned by test).
const groupPolicyTablePrefix byte = 0x20

func CouncilPolicyAddress(seq uint64) sdk.AccAddress {
	dk := make([]byte, 8)
	binary.BigEndian.PutUint64(dk, seq)
	cred, err := authtypes.NewModuleCredential(group.ModuleName, []byte{groupPolicyTablePrefix}, dk)
	if err != nil {
		panic(err)
	}
	return sdk.AccAddress(cred.Address())
}

func CouncilAuthorityBech32(prefix string) string {
	s, err := sdk.Bech32ifyAddressBytes(prefix, CouncilPolicyAddress(1))
	if err != nil {
		panic(err)
	}
	return s
}

func ValidatorBond(denom string) sdk.Coin {
	return sdk.NewCoin(denom, sdk.DefaultPowerReduction)
}

func CouncilGenesis(policyAddr string, members []string, metadata, percentage string, votingPeriod, minExecutionPeriod time.Duration, now time.Time) (*group.GenesisState, error) {
	gs := group.NewGenesisState()
	gs.GroupSeq = 1
	gs.Groups = []*group.GroupInfo{{
		Id: 1, Admin: policyAddr, Metadata: metadata, Version: 1,
		TotalWeight: strconv.Itoa(len(members)), CreatedAt: now,
	}}
	for _, m := range members {
		gs.GroupMembers = append(gs.GroupMembers, &group.GroupMember{
			GroupId: 1, Member: &group.Member{Address: m, Weight: "1", AddedAt: now},
		})
	}
	policy := &group.GroupPolicyInfo{
		Address: policyAddr, GroupId: 1, Admin: policyAddr, Metadata: metadata, Version: 1, CreatedAt: now,
	}
	if err := policy.SetDecisionPolicy(&group.PercentageDecisionPolicy{
		Percentage: percentage,
		Windows:    &group.DecisionPolicyWindows{VotingPeriod: votingPeriod, MinExecutionPeriod: minExecutionPeriod},
	}); err != nil {
		return nil, err
	}
	gs.GroupPolicySeq = 1
	gs.GroupPolicies = []*group.GroupPolicyInfo{policy}
	return gs, nil
}
