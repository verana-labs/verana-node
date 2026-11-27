package keeper_test

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/core/address"
	"cosmossdk.io/log/v2"
	"cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	"github.com/verana-labs/cosmos-group"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
	"github.com/verana-labs/verana-node/x/poa/keeper"
	"github.com/verana-labs/verana-node/x/poa/types"
)

func TestMain(m *testing.M) {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("verana", "veranapub")
	cfg.SetBech32PrefixForValidator("veranavaloper", "veranavaloperpub")
	os.Exit(m.Run())
}

const denom = "uvna"

var authority = types.CouncilAuthorityBech32("verana")

type mockStaking struct {
	params      stakingtypes.Params
	validators  map[string]stakingtypes.Validator
	delegs      []stakingtypes.Delegation
	undelegated math.LegacyDec
}

func newMockStaking() *mockStaking {
	p := stakingtypes.DefaultParams()
	p.BondDenom = denom
	p.MaxValidators = 25
	return &mockStaking{params: p, validators: map[string]stakingtypes.Validator{}}
}

func (m *mockStaking) GetParams(context.Context) (stakingtypes.Params, error) { return m.params, nil }
func (m *mockStaking) GetValidator(_ context.Context, addr sdk.ValAddress) (stakingtypes.Validator, error) {
	v, ok := m.validators[addr.String()]
	if !ok {
		return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
	}
	return v, nil
}
func (m *mockStaking) GetAllValidators(context.Context) ([]stakingtypes.Validator, error) {
	out := []stakingtypes.Validator{}
	for _, v := range m.validators {
		out = append(out, v)
	}
	return out, nil
}
func (m *mockStaking) GetAllDelegatorDelegations(context.Context, sdk.AccAddress) ([]stakingtypes.Delegation, error) {
	return m.delegs, nil
}
func (m *mockStaking) GetDelegation(_ context.Context, del sdk.AccAddress, val sdk.ValAddress) (stakingtypes.Delegation, error) {
	v, ok := m.validators[val.String()]
	if !ok || !v.DelegatorShares.IsPositive() {
		return stakingtypes.Delegation{}, stakingtypes.ErrNoDelegation
	}
	return stakingtypes.Delegation{DelegatorAddress: del.String(), ValidatorAddress: val.String(), Shares: v.DelegatorShares}, nil
}
func (m *mockStaking) Undelegate(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress, shares math.LegacyDec) (time.Time, math.Int, error) {
	m.undelegated = shares
	return time.Time{}, shares.TruncateInt(), nil
}
func (m *mockStaking) ValidatorAddressCodec() address.Codec {
	return addresscodec.NewBech32Codec("veranavaloper")
}

type mockCorporations struct{ policies map[string]bool }

func (c *mockCorporations) GetCorporationByPolicyAddress(_ context.Context, addr string) (cotypes.Corporation, error) {
	if !c.policies[addr] {
		return cotypes.Corporation{}, cotypes.ErrCorporationNotRegistered
	}
	return cotypes.Corporation{Id: 1, PolicyAddress: addr}, nil
}

type mockGroups struct {
	groupID uint64
	members []string
	windows group.DecisionPolicyWindows
}

func (g *mockGroups) GroupPolicyInfo(_ context.Context, req *group.QueryGroupPolicyInfoRequest) (*group.QueryGroupPolicyInfoResponse, error) {
	if req.Address != authority {
		return nil, errors.New("not found")
	}
	info := &group.GroupPolicyInfo{Address: authority, GroupId: g.groupID}
	windows := g.windows
	if err := info.SetDecisionPolicy(&group.PercentageDecisionPolicy{Percentage: types.CouncilPercentage, Windows: &windows}); err != nil {
		return nil, err
	}
	return &group.QueryGroupPolicyInfoResponse{Info: info}, nil
}
func (g *mockGroups) GroupMembers(context.Context, *group.QueryGroupMembersRequest) (*group.QueryGroupMembersResponse, error) {
	res := &group.QueryGroupMembersResponse{}
	for _, m := range g.members {
		res.Members = append(res.Members, &group.GroupMember{GroupId: g.groupID, Member: &group.Member{Address: m, Weight: "1"}})
	}
	return res, nil
}

type mockRouter struct{ msgs []sdk.Msg }

func (r *mockRouter) Handler(sdk.Msg) baseapp.MsgServiceHandler {
	return func(_ sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		r.msgs = append(r.msgs, msg)
		return &sdk.Result{}, nil
	}
}

type fixture struct {
	ctx     sdk.Context
	k       keeper.Keeper
	msg     types.MsgServer
	staking *mockStaking
	corps   *mockCorporations
	groups  *mockGroups
	router  *mockRouter
	member  sdk.AccAddress
	pubkey  *codectypes.Any
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	anyPk, err := codectypes.NewAnyWithValue(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)
	member := types.GroupPolicyAddress(2)
	f := &fixture{
		ctx:     sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger()),
		staking: newMockStaking(),
		corps:   &mockCorporations{policies: map[string]bool{member.String(): true}},
		groups:  &mockGroups{groupID: 1, members: []string{member.String()}, windows: group.DecisionPolicyWindows{VotingPeriod: 30 * time.Second, MinExecutionPeriod: 10 * time.Second}},
		router:  &mockRouter{},
		member:  member,
		pubkey:  anyPk,
	}
	f.k = keeper.NewKeeper(authority, addresscodec.NewBech32Codec("verana"), f.staking, f.corps, f.groups, f.router)
	f.msg = keeper.NewMsgServerImpl(f.k)
	return f
}

func (f *fixture) seat(t *testing.T, status stakingtypes.BondStatus) {
	t.Helper()
	peer := sdk.ValAddress(ed25519.GenPrivKey().PubKey().Address()).String()
	f.staking.validators[peer] = stakingtypes.Validator{OperatorAddress: peer, Status: stakingtypes.Bonded, Tokens: sdk.DefaultPowerReduction}
	valAddr := sdk.ValAddress(f.member)
	f.staking.validators[valAddr.String()] = stakingtypes.Validator{
		OperatorAddress: valAddr.String(), Status: status,
		Tokens: sdk.DefaultPowerReduction, DelegatorShares: math.LegacyNewDecFromInt(sdk.DefaultPowerReduction),
	}
}

func addMsg(f *fixture) *types.MsgAddValidator {
	return &types.MsgAddValidator{Authority: authority, ValidatorAddress: f.member.String(), Description: types.Description{Moniker: "seat"}, Pubkey: f.pubkey}
}

func TestAddValidator_RejectsNonAuthority(t *testing.T) {
	f := newFixture(t)
	msg := addMsg(f)
	msg.Authority = f.member.String()
	_, err := f.msg.AddValidator(f.ctx, msg)
	require.ErrorIs(t, err, types.ErrInvalidSigner)
}

func TestAddValidator_RejectsNonMember(t *testing.T) {
	f := newFixture(t)
	f.groups.members = nil
	_, err := f.msg.AddValidator(f.ctx, addMsg(f))
	require.ErrorIs(t, err, types.ErrNotCouncilMember)
}

func TestAddValidator_AcceptsUppercaseMember(t *testing.T) {
	f := newFixture(t)
	msg := addMsg(f)
	msg.ValidatorAddress = strings.ToUpper(msg.ValidatorAddress)
	_, err := f.msg.AddValidator(f.ctx, msg)
	require.NoError(t, err)
	require.Len(t, f.router.msgs, 1)
}

func TestAddValidator_RejectsWhenFull(t *testing.T) {
	f := newFixture(t)
	f.staking.params.MaxValidators = 1
	f.staking.validators["other"] = stakingtypes.Validator{OperatorAddress: "other", Status: stakingtypes.Bonded, Tokens: sdk.DefaultPowerReduction}
	_, err := f.msg.AddValidator(f.ctx, addMsg(f))
	require.ErrorIs(t, err, types.ErrMaxValidatorsReached)
}

func TestAddValidator_CapIgnoresUnbondingRecords(t *testing.T) {
	f := newFixture(t)
	f.staking.params.MaxValidators = 2
	f.staking.validators["bonded"] = stakingtypes.Validator{OperatorAddress: "bonded", Status: stakingtypes.Bonded, Tokens: sdk.DefaultPowerReduction}
	f.staking.validators["gone"] = stakingtypes.Validator{OperatorAddress: "gone", Status: stakingtypes.Unbonding, Jailed: true, Tokens: math.ZeroInt()}
	_, err := f.msg.AddValidator(f.ctx, addMsg(f))
	require.NoError(t, err)
	require.Len(t, f.router.msgs, 1)
}

func TestAddValidator_RejectsExistingBond(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Bonded)
	_, err := f.msg.AddValidator(f.ctx, addMsg(f))
	require.ErrorIs(t, err, types.ErrAddressHasBondedTokens)
}

func TestAddValidator_RejectsDelegations(t *testing.T) {
	f := newFixture(t)
	f.staking.delegs = []stakingtypes.Delegation{{Shares: math.LegacyOneDec()}}
	_, err := f.msg.AddValidator(f.ctx, addMsg(f))
	require.ErrorIs(t, err, types.ErrAddressHasDelegations)
}

func TestAddValidator_RejectsMissingPubkey(t *testing.T) {
	f := newFixture(t)
	msg := addMsg(f)
	msg.Pubkey = nil
	_, err := f.msg.AddValidator(f.ctx, msg)
	require.Error(t, err)
	require.Empty(t, f.router.msgs)
}

func TestAddValidator_RejectsEmptyDescription(t *testing.T) {
	f := newFixture(t)
	msg := addMsg(f)
	msg.Description = types.Description{}
	_, err := f.msg.AddValidator(f.ctx, msg)
	require.Error(t, err)
	require.Empty(t, f.router.msgs)
}

func TestAddValidator_StakesFixedBondFromOperator(t *testing.T) {
	f := newFixture(t)
	_, err := f.msg.AddValidator(f.ctx, addMsg(f))
	require.NoError(t, err)

	require.Len(t, f.router.msgs, 1)
	create, ok := f.router.msgs[0].(*stakingtypes.MsgCreateValidator)
	require.True(t, ok)
	require.Equal(t, types.ValidatorBond(denom), create.Value)
	require.Equal(t, sdk.ValAddress(f.member).String(), create.ValidatorAddress)
	require.Equal(t, "seat", create.Description.Moniker)
	require.True(t, create.MinSelfDelegation.Equal(math.OneInt()))
	require.True(t, create.Commission.Rate.IsZero())
	require.True(t, create.Commission.MaxRate.IsZero())
}

func TestRemoveValidator_RejectsUnknown(t *testing.T) {
	f := newFixture(t)
	_, err := f.msg.RemoveValidator(f.ctx, &types.MsgRemoveValidator{Authority: authority, ValidatorAddress: f.member.String()})
	require.ErrorIs(t, err, types.ErrNotAValidator)
}

func TestRemoveValidator_RejectsAlreadyRemoved(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Unbonding)
	valAddr := sdk.ValAddress(f.member).String()
	f.staking.validators[valAddr] = stakingtypes.Validator{OperatorAddress: valAddr, Status: stakingtypes.Unbonding, Tokens: math.ZeroInt(), DelegatorShares: math.LegacyZeroDec()}
	_, err := f.msg.RemoveValidator(f.ctx, &types.MsgRemoveValidator{Authority: authority, ValidatorAddress: f.member.String()})
	require.ErrorIs(t, err, types.ErrNotAValidator)
	require.True(t, f.staking.undelegated.IsNil())
}

func TestRemoveValidator_RejectsNonAuthority(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Bonded)
	_, err := f.msg.RemoveValidator(f.ctx, &types.MsgRemoveValidator{Authority: f.member.String(), ValidatorAddress: f.member.String()})
	require.ErrorIs(t, err, types.ErrInvalidSigner)
}

func TestRemoveValidator_UndelegatesSelfBond(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Bonded)
	_, err := f.msg.RemoveValidator(f.ctx, &types.MsgRemoveValidator{Authority: authority, ValidatorAddress: f.member.String()})
	require.NoError(t, err)
	require.True(t, f.staking.undelegated.Equal(math.LegacyNewDecFromInt(sdk.DefaultPowerReduction)))
}

func TestRemoveValidator_UndelegatesWhenJailed(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Unbonding)
	_, err := f.msg.RemoveValidator(f.ctx, &types.MsgRemoveValidator{Authority: authority, ValidatorAddress: f.member.String()})
	require.NoError(t, err)
	require.True(t, f.staking.undelegated.Equal(math.LegacyNewDecFromInt(sdk.DefaultPowerReduction)))
}

func TestSelfRemoveValidator_UsesSignerAddress(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Bonded)
	_, err := f.msg.SelfRemoveValidator(f.ctx, &types.MsgSelfRemoveValidator{ValidatorAddress: f.member.String()})
	require.NoError(t, err)
	require.True(t, f.staking.undelegated.Equal(math.LegacyNewDecFromInt(sdk.DefaultPowerReduction)))
}

func TestCouncilQuery_ListsMembersWithoutBondedValidator(t *testing.T) {
	f := newFixture(t)
	other := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()).String()
	f.groups.members = append(f.groups.members, other)
	f.seat(t, stakingtypes.Bonded)

	res, err := keeper.NewQueryServerImpl(f.k).Council(f.ctx, &types.QueryCouncilRequest{})
	require.NoError(t, err)
	require.Equal(t, authority, res.Authority)
	require.EqualValues(t, 1, res.GroupId)
	require.Equal(t, []string{f.member.String(), other}, res.Members)
	require.Equal(t, []string{other}, res.MembersWithoutValidator)
}
