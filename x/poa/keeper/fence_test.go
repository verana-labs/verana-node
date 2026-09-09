package keeper_test

import (
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/poa/keeper"
	"github.com/verana-labs/verana-node/x/poa/types"
)

func anyOf(t *testing.T, m sdk.Msg) *codectypes.Any {
	t.Helper()
	a, err := codectypes.NewAnyWithValue(m)
	require.NoError(t, err)
	return a
}

func check(f *fixture, inGenesis bool, msgs ...sdk.Msg) error {
	return f.k.CheckMessages(f.ctx, msgs, inGenesis)
}

func TestCheckMessages_BlocksDelegationEverywhere(t *testing.T) {
	f := newFixture(t)
	for _, msg := range []sdk.Msg{&stakingtypes.MsgDelegate{}, &stakingtypes.MsgUndelegate{}, &stakingtypes.MsgBeginRedelegate{}, &stakingtypes.MsgCancelUnbondingDelegation{}} {
		require.ErrorIs(t, check(f, false, msg), types.ErrMsgNotAllowed, sdk.MsgTypeURL(msg))
	}
	exec := &authz.MsgExec{Msgs: []*codectypes.Any{anyOf(t, &stakingtypes.MsgDelegate{})}}
	require.ErrorIs(t, check(f, false, exec), types.ErrMsgNotAllowed)
	prop := &group.MsgSubmitProposal{Messages: []*codectypes.Any{anyOf(t, exec)}}
	require.ErrorIs(t, check(f, false, prop), types.ErrMsgNotAllowed)
	require.NoError(t, check(f, false, &banktypes.MsgSend{}, &stakingtypes.MsgEditValidator{}))
}

func TestCheckMessages_CreateValidatorOnlyAtGenesisFromMemberWithFixedBond(t *testing.T) {
	f := newFixture(t)
	valoper := sdk.ValAddress(f.member).String()
	ok := &stakingtypes.MsgCreateValidator{ValidatorAddress: valoper, Value: types.ValidatorBond(denom)}
	require.NoError(t, check(f, true, ok))
	require.ErrorIs(t, check(f, false, ok), types.ErrMsgNotAllowed)
	require.ErrorIs(t, check(f, false, &group.MsgSubmitProposal{Messages: []*codectypes.Any{anyOf(t, ok)}}), types.ErrMsgNotAllowed)
	require.ErrorIs(t, check(f, true, &stakingtypes.MsgCreateValidator{ValidatorAddress: valoper, Value: sdk.NewInt64Coin(denom, 2_000_000)}), types.ErrInvalidBond)
	require.ErrorIs(t, check(f, true, &stakingtypes.MsgCreateValidator{ValidatorAddress: valoper, Value: sdk.NewInt64Coin("stake", 1_000_000)}), types.ErrInvalidBond)
	f.groups.members = nil
	require.ErrorIs(t, check(f, true, ok), types.ErrNotCouncilMember)
}

func TestCheckMessages_CouncilAdminMustStayThePolicy(t *testing.T) {
	f := newFixture(t)
	other := f.member.String()
	require.ErrorIs(t, check(f, false, &group.MsgUpdateGroupAdmin{GroupId: 1, NewAdmin: other}), types.ErrCouncilIntegrity)
	require.NoError(t, check(f, false, &group.MsgUpdateGroupAdmin{GroupId: 1, NewAdmin: authority}))
	require.NoError(t, check(f, false, &group.MsgUpdateGroupAdmin{GroupId: 2, NewAdmin: other}))
	require.ErrorIs(t, check(f, false, &group.MsgUpdateGroupPolicyAdmin{GroupPolicyAddress: authority, NewAdmin: other}), types.ErrCouncilIntegrity)
	require.NoError(t, check(f, false, &group.MsgUpdateGroupPolicyAdmin{GroupPolicyAddress: other, NewAdmin: other}))
}

func TestCheckMessages_CouncilRuleCannotDropBelowTwoThirds(t *testing.T) {
	f := newFixture(t)
	policy := func(p group.DecisionPolicy) *group.MsgUpdateGroupPolicyDecisionPolicy {
		m := &group.MsgUpdateGroupPolicyDecisionPolicy{Admin: authority, GroupPolicyAddress: authority}
		require.NoError(t, m.SetDecisionPolicy(p))
		return m
	}
	windows := &group.DecisionPolicyWindows{}
	require.ErrorIs(t, check(f, false, policy(&group.PercentageDecisionPolicy{Percentage: "0.2", Windows: windows})), types.ErrCouncilIntegrity)
	require.ErrorIs(t, check(f, false, policy(&group.ThresholdDecisionPolicy{Threshold: "1", Windows: windows})), types.ErrCouncilIntegrity)
	require.NoError(t, check(f, false, policy(&group.PercentageDecisionPolicy{Percentage: "0.75", Windows: windows})))
	require.NoError(t, check(f, false, policy(&group.PercentageDecisionPolicy{Percentage: types.CouncilPercentage, Windows: windows})))
	other := policy(&group.PercentageDecisionPolicy{Percentage: "0.1", Windows: windows})
	other.GroupPolicyAddress = f.member.String()
	require.NoError(t, check(f, false, other))
}

func TestCheckMessages_CouncilSeatsStayWeightOneAndNonEmpty(t *testing.T) {
	f := newFixture(t)
	member := f.member.String()
	newcomer := sdk.AccAddress([]byte("newcomer____________")).String()
	weighted := &group.MsgUpdateGroupMembers{GroupId: 1, MemberUpdates: []group.MemberRequest{{Address: newcomer, Weight: "3"}}}
	require.ErrorIs(t, check(f, false, weighted), types.ErrCouncilIntegrity)
	emptied := &group.MsgUpdateGroupMembers{GroupId: 1, MemberUpdates: []group.MemberRequest{{Address: member, Weight: "0"}}}
	require.ErrorIs(t, check(f, false, emptied), types.ErrCouncilIntegrity)
	swap := &group.MsgUpdateGroupMembers{GroupId: 1, MemberUpdates: []group.MemberRequest{{Address: member, Weight: "0"}, {Address: newcomer, Weight: "1"}}}
	require.NoError(t, check(f, false, swap))
	otherGroup := &group.MsgUpdateGroupMembers{GroupId: 2, MemberUpdates: []group.MemberRequest{{Address: newcomer, Weight: "3"}}}
	require.NoError(t, check(f, false, otherGroup))
	require.ErrorIs(t, check(f, false, &group.MsgLeaveGroup{GroupId: 1, Address: member}), types.ErrCouncilIntegrity)
	f.groups.members = append(f.groups.members, newcomer)
	require.NoError(t, check(f, false, &group.MsgLeaveGroup{GroupId: 1, Address: member}))
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
