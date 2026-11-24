package keeper_test

import (
	"strings"
	"testing"
	"time"

	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"github.com/verana-labs/cosmos-group"

	"github.com/verana-labs/verana-node/x/poa/keeper"
	"github.com/verana-labs/verana-node/x/poa/types"
)

func anyOf(t *testing.T, m sdk.Msg) *codectypes.Any {
	t.Helper()
	a, err := codectypes.NewAnyWithValue(m)
	require.NoError(t, err)
	return a
}

func check(f *fixture, msgs ...sdk.Msg) error {
	return f.k.CheckMessages(f.ctx, msgs)
}

func TestCheckMessages_BlocksDelegationEverywhere(t *testing.T) {
	f := newFixture(t)
	for _, msg := range []sdk.Msg{&stakingtypes.MsgDelegate{}, &stakingtypes.MsgUndelegate{}, &stakingtypes.MsgBeginRedelegate{}, &stakingtypes.MsgCancelUnbondingDelegation{}} {
		require.ErrorIs(t, check(f, msg), types.ErrMsgNotAllowed, sdk.MsgTypeURL(msg))
	}
	exec := &authz.MsgExec{Msgs: []*codectypes.Any{anyOf(t, &stakingtypes.MsgDelegate{})}}
	require.ErrorIs(t, check(f, exec), types.ErrMsgNotAllowed)
	prop := &group.MsgSubmitProposal{Messages: []*codectypes.Any{anyOf(t, exec)}}
	require.ErrorIs(t, check(f, prop), types.ErrMsgNotAllowed)
	require.NoError(t, check(f, &banktypes.MsgSend{}, &stakingtypes.MsgEditValidator{}))
}

func TestCheckMessages_RefusesCreateValidator(t *testing.T) {
	f := newFixture(t)
	create := &stakingtypes.MsgCreateValidator{ValidatorAddress: sdk.ValAddress(f.member).String(), Value: types.ValidatorBond(denom)}
	require.ErrorIs(t, check(f, create), types.ErrMsgNotAllowed)
	require.ErrorIs(t, check(f, &group.MsgSubmitProposal{Messages: []*codectypes.Any{anyOf(t, create)}}), types.ErrMsgNotAllowed)
}

func TestCheckMessages_CouncilAdminMustStayThePolicy(t *testing.T) {
	f := newFixture(t)
	other := f.member.String()
	require.ErrorIs(t, check(f, &group.MsgUpdateGroupAdmin{GroupId: 1, NewAdmin: other}), types.ErrCouncilIntegrity)
	require.NoError(t, check(f, &group.MsgUpdateGroupAdmin{GroupId: 1, NewAdmin: authority}))
	require.NoError(t, check(f, &group.MsgUpdateGroupAdmin{GroupId: 2, NewAdmin: other}))
	require.ErrorIs(t, check(f, &group.MsgUpdateGroupPolicyAdmin{GroupPolicyAddress: authority, NewAdmin: other}), types.ErrCouncilIntegrity)
	require.NoError(t, check(f, &group.MsgUpdateGroupPolicyAdmin{GroupPolicyAddress: other, NewAdmin: other}))
}

func TestCheckMessages_CouncilRuleCannotDropBelowTwoThirds(t *testing.T) {
	f := newFixture(t)
	policy := func(p group.DecisionPolicy) *group.MsgUpdateGroupPolicyDecisionPolicy {
		m := &group.MsgUpdateGroupPolicyDecisionPolicy{Admin: authority, GroupPolicyAddress: authority}
		require.NoError(t, m.SetDecisionPolicy(p))
		return m
	}
	windows := &group.DecisionPolicyWindows{VotingPeriod: 30 * time.Second, MinExecutionPeriod: 10 * time.Second}
	require.ErrorIs(t, check(f, policy(&group.PercentageDecisionPolicy{Percentage: "0.2", Windows: windows})), types.ErrCouncilIntegrity)
	require.ErrorIs(t, check(f, policy(&group.ThresholdDecisionPolicy{Threshold: "1", Windows: windows})), types.ErrCouncilIntegrity)
	require.NoError(t, check(f, policy(&group.PercentageDecisionPolicy{Percentage: "0.75", Windows: windows})))
	require.NoError(t, check(f, policy(&group.PercentageDecisionPolicy{Percentage: types.CouncilPercentage, Windows: windows})))
	other := policy(&group.PercentageDecisionPolicy{Percentage: "0.1", Windows: windows})
	other.GroupPolicyAddress = f.member.String()
	require.NoError(t, check(f, other))
}

func TestCheckMessages_CouncilWindowsCannotShrink(t *testing.T) {
	f := newFixture(t)
	policy := func(voting, minExec time.Duration) *group.MsgUpdateGroupPolicyDecisionPolicy {
		m := &group.MsgUpdateGroupPolicyDecisionPolicy{Admin: authority, GroupPolicyAddress: authority}
		require.NoError(t, m.SetDecisionPolicy(&group.PercentageDecisionPolicy{Percentage: types.CouncilPercentage, Windows: &group.DecisionPolicyWindows{VotingPeriod: voting, MinExecutionPeriod: minExec}}))
		return m
	}
	require.ErrorIs(t, check(f, policy(time.Second, 10*time.Second)), types.ErrCouncilIntegrity)
	require.ErrorIs(t, check(f, policy(30*time.Second, 0)), types.ErrCouncilIntegrity)
	require.NoError(t, check(f, policy(30*time.Second, 10*time.Second)))
	require.NoError(t, check(f, policy(time.Hour, time.Minute)))

	noWindows := &group.MsgUpdateGroupPolicyDecisionPolicy{Admin: authority, GroupPolicyAddress: authority}
	require.NoError(t, noWindows.SetDecisionPolicy(&group.PercentageDecisionPolicy{Percentage: types.CouncilPercentage}))
	require.ErrorIs(t, check(f, noWindows), types.ErrCouncilIntegrity)
}

func TestCheckMessages_SeatDropsOnlyWithItsValidator(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Bonded)
	member := f.member.String()
	f.groups.members = append(f.groups.members, types.GroupPolicyAddress(3).String())

	drop := &group.MsgUpdateGroupMembers{GroupId: 1, MemberUpdates: []group.MemberRequest{{Address: member, Weight: "0"}}}
	require.ErrorIs(t, check(f, drop), types.ErrCouncilIntegrity)
	require.NoError(t, check(f, drop, &types.MsgRemoveValidator{Authority: authority, ValidatorAddress: member}))
	require.NoError(t, check(f, &group.MsgSubmitProposal{Messages: []*codectypes.Any{anyOf(t, drop), anyOf(t, &types.MsgRemoveValidator{Authority: authority, ValidatorAddress: member})}}))

	f.staking.validators[sdk.ValAddress(f.member).String()] = stakingtypes.Validator{OperatorAddress: sdk.ValAddress(f.member).String(), Status: stakingtypes.Unbonding, Tokens: math.ZeroInt()}
	require.NoError(t, check(f, drop))
}

func TestCheckMessages_RefusesCouncilLeave(t *testing.T) {
	f := newFixture(t)
	member := f.member.String()
	f.groups.members = append(f.groups.members, types.GroupPolicyAddress(3).String())
	leave := &group.MsgLeaveGroup{GroupId: 1, Address: member}
	require.ErrorIs(t, check(f, leave), types.ErrCouncilIntegrity)
	require.ErrorIs(t, check(f, &types.MsgSelfRemoveValidator{ValidatorAddress: member}, leave), types.ErrCouncilIntegrity)
	require.ErrorIs(t, check(f, &group.MsgSubmitProposal{Messages: []*codectypes.Any{anyOf(t, leave)}}), types.ErrCouncilIntegrity)
	require.NoError(t, check(f, &group.MsgLeaveGroup{GroupId: 2, Address: member}))
}

func TestCheckMessages_CouncilMembersMustBeCorporations(t *testing.T) {
	f := newFixture(t)
	corp := types.GroupPolicyAddress(3).String()
	key := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()).String()
	f.corps.policies[corp] = true

	add := func(addr string) *group.MsgUpdateGroupMembers {
		return &group.MsgUpdateGroupMembers{GroupId: 1, MemberUpdates: []group.MemberRequest{{Address: addr, Weight: "1"}}}
	}
	require.NoError(t, check(f, add(corp)))
	require.NoError(t, check(f, add(strings.ToUpper(corp))))
	require.ErrorIs(t, check(f, add(key)), types.ErrNotACorporation)
	require.ErrorIs(t, check(f, &group.MsgSubmitProposal{Messages: []*codectypes.Any{anyOf(t, add(key))}}), types.ErrNotACorporation)
	require.NoError(t, check(f, &group.MsgUpdateGroupMembers{GroupId: 2, MemberUpdates: []group.MemberRequest{{Address: key, Weight: "1"}}}))
	require.NoError(t, check(f, &group.MsgUpdateGroupMembers{GroupId: 1, MemberUpdates: []group.MemberRequest{{Address: key, Weight: "0"}}}))
}

func TestCheckMessages_UppercaseAddressesAreStillTheCouncil(t *testing.T) {
	f := newFixture(t)
	upper := strings.ToUpper(authority)
	other := f.member.String()

	policy := &group.MsgUpdateGroupPolicyDecisionPolicy{Admin: authority, GroupPolicyAddress: upper}
	require.NoError(t, policy.SetDecisionPolicy(&group.ThresholdDecisionPolicy{Threshold: "1", Windows: &group.DecisionPolicyWindows{VotingPeriod: time.Minute, MinExecutionPeriod: time.Minute}}))
	require.ErrorIs(t, check(f, policy), types.ErrCouncilIntegrity)
	require.ErrorIs(t, check(f, &group.MsgUpdateGroupPolicyAdmin{Admin: authority, GroupPolicyAddress: upper, NewAdmin: other}), types.ErrCouncilIntegrity)
	require.NoError(t, check(f, &group.MsgUpdateGroupAdmin{GroupId: 1, NewAdmin: upper}))

	f.seat(t, stakingtypes.Bonded)
	f.groups.members = append(f.groups.members, types.GroupPolicyAddress(3).String())
	drop := &group.MsgUpdateGroupMembers{GroupId: 1, MemberUpdates: []group.MemberRequest{{Address: strings.ToUpper(other), Weight: "0"}}}
	require.ErrorIs(t, check(f, drop), types.ErrCouncilIntegrity)
	require.NoError(t, check(f, drop, &types.MsgRemoveValidator{Authority: authority, ValidatorAddress: other}))
}

func TestCheckMessages_CouncilSeatsStayWeightOneAndNonEmpty(t *testing.T) {
	f := newFixture(t)
	member := f.member.String()
	newcomer := types.GroupPolicyAddress(3).String()
	f.corps.policies[newcomer] = true
	weighted := &group.MsgUpdateGroupMembers{GroupId: 1, MemberUpdates: []group.MemberRequest{{Address: newcomer, Weight: "3"}}}
	require.ErrorIs(t, check(f, weighted), types.ErrCouncilIntegrity)
	emptied := &group.MsgUpdateGroupMembers{GroupId: 1, MemberUpdates: []group.MemberRequest{{Address: member, Weight: "0"}}}
	require.ErrorIs(t, check(f, emptied), types.ErrCouncilIntegrity)
	swap := &group.MsgUpdateGroupMembers{GroupId: 1, MemberUpdates: []group.MemberRequest{{Address: member, Weight: "0"}, {Address: newcomer, Weight: "1"}}}
	require.NoError(t, check(f, swap))
	otherGroup := &group.MsgUpdateGroupMembers{GroupId: 2, MemberUpdates: []group.MemberRequest{{Address: newcomer, Weight: "3"}}}
	require.NoError(t, check(f, otherGroup))
}

func TestRemoveValidator_RefusesTheLastBondedValidator(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Bonded)
	mine := sdk.ValAddress(f.member).String()
	for op := range f.staking.validators {
		if op != mine {
			delete(f.staking.validators, op)
		}
	}
	_, err := f.msg.RemoveValidator(f.ctx, &types.MsgRemoveValidator{Authority: authority, ValidatorAddress: f.member.String()})
	require.ErrorIs(t, err, types.ErrLastValidator)
	_, err = f.msg.SelfRemoveValidator(f.ctx, &types.MsgSelfRemoveValidator{ValidatorAddress: f.member.String()})
	require.ErrorIs(t, err, types.ErrLastValidator)
	peer := sdk.ValAddress(ed25519.GenPrivKey().PubKey().Address()).String()
	f.staking.validators[peer] = stakingtypes.Validator{OperatorAddress: peer, Status: stakingtypes.Bonded, Tokens: sdk.DefaultPowerReduction}
	_, err = f.msg.RemoveValidator(f.ctx, &types.MsgRemoveValidator{Authority: authority, ValidatorAddress: f.member.String()})
	require.NoError(t, err)
}

func TestCouncilQuery_FlagsValidatorsWithoutSeat(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Bonded)
	f.groups.members = nil
	res, err := keeper.NewQueryServerImpl(f.k).Council(f.ctx, &types.QueryCouncilRequest{})
	require.NoError(t, err)
	require.Contains(t, res.ValidatorsWithoutSeat, f.member.String())
}
