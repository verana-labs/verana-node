package keeper_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"cosmossdk.io/core/address"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

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
	params     stakingtypes.Params
	validators map[string]stakingtypes.Validator
	delegs     []stakingtypes.Delegation
	removed    math.Int
	unbonded   math.LegacyDec
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
func (m *mockStaking) RemoveValidatorTokens(_ context.Context, v stakingtypes.Validator, tokens math.Int) (stakingtypes.Validator, error) {
	m.removed = tokens
	v = v.RemoveTokens(tokens)
	m.validators[v.OperatorAddress] = v
	return v, nil
}
func (m *mockStaking) Unbond(_ context.Context, _ sdk.AccAddress, _ sdk.ValAddress, shares math.LegacyDec) (math.Int, error) {
	m.unbonded = shares
	return math.ZeroInt(), nil
}
func (m *mockStaking) Hooks() stakingtypes.StakingHooks { return stakingtypes.NewMultiStakingHooks() }
func (m *mockStaking) ValidatorAddressCodec() address.Codec {
	return addresscodec.NewBech32Codec("veranavaloper")
}

type bankCall struct {
	op     string
	module string
	amt    sdk.Coins
}

type mockBank struct{ calls []bankCall }

func (b *mockBank) MintCoins(_ context.Context, module string, amt sdk.Coins) error {
	b.calls = append(b.calls, bankCall{"mint", module, amt})
	return nil
}
func (b *mockBank) BurnCoins(_ context.Context, module string, amt sdk.Coins) error {
	b.calls = append(b.calls, bankCall{"burn", module, amt})
	return nil
}
func (b *mockBank) SendCoinsFromModuleToAccount(_ context.Context, module string, _ sdk.AccAddress, amt sdk.Coins) error {
	b.calls = append(b.calls, bankCall{"send", module, amt})
	return nil
}

type mockGroups struct {
	groupID uint64
	members []string
}

func (g *mockGroups) GroupPolicyInfo(_ context.Context, req *group.QueryGroupPolicyInfoRequest) (*group.QueryGroupPolicyInfoResponse, error) {
	if req.Address != authority {
		return nil, errors.New("not found")
	}
	return &group.QueryGroupPolicyInfoResponse{Info: &group.GroupPolicyInfo{Address: authority, GroupId: g.groupID}}, nil
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
	bank    *mockBank
	groups  *mockGroups
	router  *mockRouter
	member  sdk.AccAddress
	pubkey  *codectypes.Any
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	pk := ed25519.GenPrivKey().PubKey()
	anyPk, err := codectypes.NewAnyWithValue(pk)
	require.NoError(t, err)
	member := sdk.AccAddress(pk.Address())
	f := &fixture{
		ctx:     sdk.NewContext(nil, cmtproto.Header{}, false, log.NewNopLogger()),
		staking: newMockStaking(),
		bank:    &mockBank{},
		groups:  &mockGroups{groupID: 1, members: []string{member.String()}},
		router:  &mockRouter{},
		member:  member,
		pubkey:  anyPk,
	}
	f.k = keeper.NewKeeper(authority, addresscodec.NewBech32Codec("verana"), f.staking, f.bank, f.groups, f.router)
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

func TestAddValidator_RejectsWhenFull(t *testing.T) {
	f := newFixture(t)
	f.staking.params.MaxValidators = 1
	f.staking.validators["other"] = stakingtypes.Validator{OperatorAddress: "other"}
	_, err := f.msg.AddValidator(f.ctx, addMsg(f))
	require.ErrorIs(t, err, types.ErrMaxValidatorsReached)
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

func TestAddValidator_RejectsMissingPubkeyBeforeMinting(t *testing.T) {
	f := newFixture(t)
	msg := addMsg(f)
	msg.Pubkey = nil
	_, err := f.msg.AddValidator(f.ctx, msg)
	require.Error(t, err)
	require.Empty(t, f.bank.calls)
}

func TestAddValidator_RejectsEmptyDescriptionBeforeMinting(t *testing.T) {
	f := newFixture(t)
	msg := addMsg(f)
	msg.Description = types.Description{}
	_, err := f.msg.AddValidator(f.ctx, msg)
	require.Error(t, err)
	require.Empty(t, f.bank.calls)
}

func TestAddValidator_MintsFixedBondAndRoutesCreate(t *testing.T) {
	f := newFixture(t)
	_, err := f.msg.AddValidator(f.ctx, addMsg(f))
	require.NoError(t, err)

	bond := sdk.NewCoins(types.ValidatorBond(denom))
	require.Equal(t, []bankCall{{"mint", types.ModuleName, bond}, {"send", types.ModuleName, bond}}, f.bank.calls)

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

func TestRemoveValidator_RejectsNonAuthority(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Bonded)
	_, err := f.msg.RemoveValidator(f.ctx, &types.MsgRemoveValidator{Authority: f.member.String(), ValidatorAddress: f.member.String()})
	require.ErrorIs(t, err, types.ErrInvalidSigner)
}

func TestRemoveValidator_BurnsFromBondedPool(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Bonded)
	_, err := f.msg.RemoveValidator(f.ctx, &types.MsgRemoveValidator{Authority: authority, ValidatorAddress: f.member.String()})
	require.NoError(t, err)

	bond := sdk.NewCoins(types.ValidatorBond(denom))
	require.Equal(t, []bankCall{{"burn", stakingtypes.BondedPoolName, bond}}, f.bank.calls)
	require.True(t, f.staking.removed.Equal(sdk.DefaultPowerReduction))
	require.True(t, f.staking.unbonded.Equal(math.LegacyNewDecFromInt(sdk.DefaultPowerReduction)))
}

func TestRemoveValidator_BurnsFromNotBondedPoolWhenJailed(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Unbonding)
	_, err := f.msg.RemoveValidator(f.ctx, &types.MsgRemoveValidator{Authority: authority, ValidatorAddress: f.member.String()})
	require.NoError(t, err)
	require.Equal(t, stakingtypes.NotBondedPoolName, f.bank.calls[0].module)
}

func TestSelfRemoveValidator_UsesSignerAddress(t *testing.T) {
	f := newFixture(t)
	f.seat(t, stakingtypes.Bonded)
	_, err := f.msg.SelfRemoveValidator(f.ctx, &types.MsgSelfRemoveValidator{ValidatorAddress: f.member.String()})
	require.NoError(t, err)
	require.True(t, f.staking.removed.Equal(sdk.DefaultPowerReduction))
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
