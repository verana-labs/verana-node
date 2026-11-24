package app_test

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/log/v2"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	gogoproto "github.com/cosmos/gogoproto/proto"
	icagenesistypes "github.com/cosmos/ibc-go/v11/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v11/modules/apps/27-interchain-accounts/types"
	"github.com/stretchr/testify/require"
	"github.com/verana-labs/cosmos-group"

	"github.com/verana-labs/verana-node/app"
	cotypes "github.com/verana-labs/verana-node/x/co/types"
	gftypes "github.com/verana-labs/verana-node/x/gf/types"
	poakeeper "github.com/verana-labs/verana-node/x/poa/keeper"
	poatypes "github.com/verana-labs/verana-node/x/poa/types"
)

func newTestApp(t *testing.T) *app.App {
	t.Helper()
	a, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("council-test"))
	require.NoError(t, err)
	return a
}

// corp is a genesis corporation with one signing member key.
type corp struct {
	key    *secp256k1.PrivKey
	policy sdk.AccAddress
	cons   cryptotypes.PrivKey
}

func (c corp) member() sdk.AccAddress { return sdk.AccAddress(c.key.PubKey().Address()) }

const memberFunds = 10_000_000

// councilGenesis seats n corporations; the first runs the only genesis validator and every policy holds one bond.
func councilGenesis(t *testing.T, a *app.App, n int) (map[string]json.RawMessage, []corp) {
	t.Helper()
	gs := a.DefaultGenesis()
	var accs []authtypes.GenesisAccount
	var bals []banktypes.Balance
	spec := app.GenesisCouncil{
		Percentage: poatypes.CouncilPercentage, VotingPeriod: time.Hour, Metadata: "test council", MaxValidators: 25,
	}
	corps := make([]corp, n)
	for i := range corps {
		corps[i] = corp{key: secp256k1.GenPrivKey(), policy: poatypes.GroupPolicyAddress(uint64(i + 2))}
		member := corps[i].member()
		accs = append(accs, authtypes.NewBaseAccount(member, nil, uint64(i+1), 0))
		bals = append(bals, banktypes.Balance{Address: member.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, memberFunds))})
		gc := app.GenesisCorporation{
			Did: fmt.Sprintf("did:example:corp%d", i), Language: "en", Members: []string{member.String()},
			Funds: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.DefaultPowerReduction)),
		}
		if i == 0 {
			corps[i].cons = ed25519.GenPrivKey()
			gc.Validator = &app.GenesisValidator{Moniker: "node0", PubKey: corps[i].cons.PubKey()}
		}
		spec.Corporations = append(spec.Corporations, gc)
	}
	authGen := authtypes.NewGenesisState(authtypes.DefaultParams(), accs)
	gs[authtypes.ModuleName] = a.AppCodec().MustMarshalJSON(authGen)
	bankGen := banktypes.DefaultGenesisState()
	bankGen.Balances = bals
	gs[banktypes.ModuleName] = a.AppCodec().MustMarshalJSON(bankGen)
	require.NoError(t, app.SetGenesisCouncil(a.AppCodec(), gs, spec, time.Unix(0, 0).UTC()))
	return gs, corps
}

func initChain(t *testing.T, a *app.App, gs map[string]json.RawMessage) error {
	t.Helper()
	bz, err := json.Marshal(gs)
	require.NoError(t, err)
	_, err = a.InitChain(&abci.RequestInitChain{
		ChainId: "council-test", AppStateBytes: bz,
		ConsensusParams: simtestutil.DefaultConsensusParams, Time: time.Unix(0, 0).UTC(),
	})
	return err
}

func mutate[T any, PT interface {
	*T
	gogoproto.Message
}](t *testing.T, a *app.App, gs map[string]json.RawMessage, module string, fn func(PT)) {
	t.Helper()
	state := PT(new(T))
	require.NoError(t, a.AppCodec().UnmarshalJSON(gs[module], state))
	fn(state)
	bz, err := a.AppCodec().MarshalJSON(state)
	require.NoError(t, err)
	gs[module] = bz
}

func mutateCouncil(t *testing.T, a *app.App, gs map[string]json.RawMessage, fn func(*group.GenesisState)) {
	t.Helper()
	mutate(t, a, gs, group.ModuleName, fn)
}

// dropFromCouncil removes a seat from the council group in genesis.
func dropFromCouncil(t *testing.T, a *app.App, gs map[string]json.RawMessage, addr string) {
	t.Helper()
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) {
		kept := cg.GroupMembers[:0]
		for _, m := range cg.GroupMembers {
			if m.GroupId == 1 && m.Member.Address == addr {
				continue
			}
			kept = append(kept, m)
		}
		cg.GroupMembers = kept
		n := 0
		for _, m := range kept {
			if m.GroupId == 1 {
				n++
			}
		}
		cg.Groups[0].TotalWeight = fmt.Sprint(n)
	})
}

func TestInitChain_RequiresCouncil(t *testing.T) {
	a := newTestApp(t)
	gs, _ := councilGenesis(t, a, 3)
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) {
		cg.GroupPolicies = cg.GroupPolicies[1:]
	})
	require.ErrorContains(t, initChain(t, a, gs), "no council group policy")
}

func TestInitChain_AcceptsCouncilOfCorporations(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	require.NoError(t, initChain(t, a, gs))
	(&liveChain{app: a, now: time.Unix(0, 0).UTC()}).block(t)

	ctx := a.NewContext(true)
	members, err := a.PoaKeeper.CouncilMembers(ctx)
	require.NoError(t, err)
	require.Len(t, members, 3)
	for i, c := range corps {
		co, err := a.CoKeeper.ResolveCorporationByPolicyAddress(ctx, c.policy.String())
		require.NoError(t, err)
		require.EqualValues(t, i+1, co.Id)
		info, err := a.GroupKeeper.GroupPolicyInfo(ctx, &group.QueryGroupPolicyInfoRequest{Address: c.policy.String()})
		require.NoError(t, err)
		require.Equal(t, c.policy.String(), info.Info.Admin)
	}
	v, err := a.StakingKeeper.GetValidator(ctx, sdk.ValAddress(corps[0].policy))
	require.NoError(t, err)
	require.True(t, v.IsBonded())
	require.True(t, v.Tokens.Equal(sdk.DefaultPowerReduction))
}

func TestInitChain_RequiresThreeCorporations(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	dropFromCouncil(t, a, gs, corps[2].policy.String())
	require.ErrorContains(t, initChain(t, a, gs), "at least 3")
}

func TestInitChain_RejectsMemberWithoutCorporation(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 4)
	dropFromCouncil(t, a, gs, corps[3].policy.String())
	mutate(t, a, gs, cotypes.ModuleName, func(co *cotypes.GenesisState) {
		co.Corporations = co.Corporations[:3]
		co.Corporations[2].PolicyAddress = corps[3].policy.String()
	})
	require.ErrorContains(t, initChain(t, a, gs), "is not a corporation")
}

func TestInitChain_RejectsCorporationWithoutPolicy(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	key := corps[2].member().String()
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) {
		for _, m := range cg.GroupMembers {
			if m.GroupId == 1 && m.Member.Address == corps[2].policy.String() {
				m.Member.Address = key
			}
		}
	})
	mutate(t, a, gs, cotypes.ModuleName, func(co *cotypes.GenesisState) {
		co.Corporations[2].PolicyAddress = key
	})
	require.ErrorContains(t, initChain(t, a, gs), "has no group policy")
}

func TestInitChain_RejectsCorporationNotAdministeringItsPolicy(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) {
		for _, p := range cg.GroupPolicies {
			if p.Address == corps[1].policy.String() {
				p.Admin = corps[1].member().String()
			}
		}
	})
	require.ErrorContains(t, initChain(t, a, gs), "must administer its own policy")
}

func TestInitChain_RejectsValidatorOutsideCouncil(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 4)
	dropFromCouncil(t, a, gs, corps[0].policy.String())
	require.ErrorContains(t, initChain(t, a, gs), "not a council member")
}

func TestInitChain_RejectsOpenICAHost(t *testing.T) {
	a := newTestApp(t)
	gs, _ := councilGenesis(t, a, 3)
	mutate(t, a, gs, icatypes.ModuleName, func(ica *icagenesistypes.GenesisState) {
		ica.HostGenesisState.Params.AllowMessages = []string{"*"}
	})
	require.ErrorContains(t, initChain(t, a, gs), "ica host allow_messages")
}

func TestInitChain_RejectsForeignPolicyAdmin(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) { cg.GroupPolicies[0].Admin = corps[0].policy.String() })
	require.ErrorContains(t, initChain(t, a, gs), "policy admin must be the policy")
}

func TestInitChain_RejectsWeightedMember(t *testing.T) {
	a := newTestApp(t)
	gs, _ := councilGenesis(t, a, 3)
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) { cg.GroupMembers[0].Member.Weight = "2"; cg.Groups[0].TotalWeight = "4" })
	require.ErrorContains(t, initChain(t, a, gs), "has weight 2")
}

func prebake(t *testing.T, cg *group.GenesisState, status group.ProposalStatus, msg sdk.Msg) {
	t.Helper()
	now := time.Unix(0, 0).UTC()
	p := &group.Proposal{
		Id: 1, GroupPolicyAddress: app.CouncilAuthority, Proposers: []string{cg.GroupMembers[0].Member.Address},
		SubmitTime: now, GroupVersion: 1, GroupPolicyVersion: 1, Status: status,
		VotingPeriodEnd: now.Add(time.Hour), ExecutorResult: group.PROPOSAL_EXECUTOR_RESULT_NOT_RUN, Title: "t", Summary: "s",
		FinalTallyResult: group.TallyResult{YesCount: "0", NoCount: "0", AbstainCount: "0", NoWithVetoCount: "0"},
	}
	require.NoError(t, p.SetMsgs([]sdk.Msg{msg}))
	cg.ProposalSeq = 1
	cg.Proposals = []*group.Proposal{p}
}

func TestInitChain_RejectsPrebakedFencedProposal(t *testing.T) {
	a := newTestApp(t)
	gs, _ := councilGenesis(t, a, 3)
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) {
		prebake(t, cg, group.PROPOSAL_STATUS_SUBMITTED, &stakingtypes.MsgDelegate{DelegatorAddress: app.CouncilAuthority, ValidatorAddress: "v", Amount: sdk.NewInt64Coin("stake", 1)})
	})
	require.ErrorContains(t, initChain(t, a, gs), "not allowed")
}

func TestInitChain_KeepsHistoricalProposals(t *testing.T) {
	a := newTestApp(t)
	gs, _ := councilGenesis(t, a, 3)
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) {
		prebake(t, cg, group.PROPOSAL_STATUS_REJECTED, &stakingtypes.MsgDelegate{DelegatorAddress: app.CouncilAuthority, ValidatorAddress: "v", Amount: sdk.NewInt64Coin("stake", 1)})
	})
	require.NoError(t, initChain(t, a, gs))
}

// GOV-1 seats a corporation through the AUTHZ-CHECK-5 index alone: a lapsed
// network framework acceptance stops the corporation's own acts, not its seat.
func TestInitChain_SeatIgnoresNetworkFrameworkAcceptance(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	at := time.Unix(0, 0).UTC()
	mutate(t, a, gs, gftypes.ModuleName, func(g *gftypes.GenesisState) {
		g.Versions = append(g.Versions, gftypes.GovernanceFrameworkVersion{Id: 2, Created: at, Version: 2, ActiveSince: &at})
		g.Documents = append(g.Documents, gftypes.GovernanceFrameworkDocument{
			Id: 2, GfvId: 2, Created: at, Language: "en", Url: gftypes.DefaultNGFDocURL, DigestSri: gftypes.DefaultNGFDocDigestSRI,
		})
		g.GfvCounter, g.GfdCounter = 2, 2
		g.NetworkGovernance = gftypes.NetworkGovernance{NetworkGovernanceFrameworkVersions: []uint64{1, 2}, NgfAcceptanceDeadline: &at}
	})
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) {
		prebake(t, cg, group.PROPOSAL_STATUS_SUBMITTED, &group.MsgUpdateGroupMembers{
			Admin: app.CouncilAuthority, GroupId: 1,
			MemberUpdates: []group.MemberRequest{{Address: corps[1].policy.String(), Weight: "1"}},
		})
	})
	require.NoError(t, initChain(t, a, gs))
	(&liveChain{app: a, now: at}).block(t)

	ctx := a.NewContext(true).WithBlockTime(at.Add(time.Second))
	_, err := a.CoKeeper.ResolveCorporationByPolicyAddress(ctx, corps[1].policy.String())
	require.ErrorIs(t, err, gftypes.ErrGFNotAccepted)
}

func TestGenesisValidator_SignsBlocks(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	require.NoError(t, initChain(t, a, gs))
	c := &liveChain{app: a, now: time.Unix(0, 0).UTC()}
	c.block(t)

	consAddr := sdk.ConsAddress(corps[0].cons.PubKey().Address())
	c.height++
	c.now = c.now.Add(time.Second)
	_, err := a.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: c.height, Time: c.now,
		DecidedLastCommit: abci.CommitInfo{Votes: []abci.VoteInfo{{
			Validator: abci.Validator{Address: consAddr, Power: 1}, BlockIdFlag: cmtproto.BlockIDFlagCommit,
		}}},
	})
	require.NoError(t, err)
	_, err = a.Commit()
	require.NoError(t, err)
	info, err := a.SlashingKeeper.GetValidatorSigningInfo(a.NewContext(true), consAddr)
	require.NoError(t, err)
	require.Equal(t, consAddr.String(), info.Address)
}

func TestSeatAndUnseat_ThroughStaking(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	require.NoError(t, initChain(t, a, gs))
	_, err := a.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: time.Unix(1, 0).UTC()})
	require.NoError(t, err)
	_, err = a.Commit()
	require.NoError(t, err)

	ctx := a.NewUncachedContext(false, cmtproto.Header{Height: 2, Time: time.Unix(2, 0).UTC()})
	supply0 := a.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom)
	member := corps[1].policy
	pk, err := codectypes.NewAnyWithValue(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)
	srv := poakeeper.NewMsgServerImpl(a.PoaKeeper)

	_, err = srv.AddValidator(ctx, &poatypes.MsgAddValidator{Authority: app.CouncilAuthority, ValidatorAddress: member.String(), Description: poatypes.Description{Moniker: "m"}, Pubkey: pk})
	require.NoError(t, err)
	require.True(t, a.BankKeeper.GetBalance(ctx, member, sdk.DefaultBondDenom).IsZero())
	require.True(t, supply0.Equal(a.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom)))
	updates, err := a.StakingKeeper.EndBlocker(ctx)
	require.NoError(t, err)
	require.Len(t, updates, 1)
	require.EqualValues(t, 1, updates[0].Power)

	_, err = srv.RemoveValidator(ctx, &poatypes.MsgRemoveValidator{Authority: app.CouncilAuthority, ValidatorAddress: member.String()})
	require.NoError(t, err)
	updates, err = a.StakingKeeper.EndBlocker(ctx)
	require.NoError(t, err)
	require.Len(t, updates, 1)
	require.EqualValues(t, 0, updates[0].Power)
	require.True(t, supply0.Equal(a.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom)))

	v, err := a.StakingKeeper.GetValidator(ctx, sdk.ValAddress(member))
	require.NoError(t, err)
	require.True(t, v.IsUnbonding(), v.Status.String())
	ubd, err := a.StakingKeeper.GetUnbondingDelegation(ctx, member, sdk.ValAddress(member))
	require.NoError(t, err)
	require.True(t, ubd.Entries[0].Balance.Equal(sdk.DefaultPowerReduction))
}

func TestReseatAfterUnbonding_ReturnsBondAndKeepsRecordUntilMature(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	require.NoError(t, initChain(t, a, gs))
	(&liveChain{app: a, now: time.Unix(0, 0).UTC()}).block(t)

	member := corps[1].policy
	valAddr := sdk.ValAddress(member)
	pk, err := codectypes.NewAnyWithValue(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)
	srv := poakeeper.NewMsgServerImpl(a.PoaKeeper)
	block := func(height int64, at time.Time, fn func(ctx sdk.Context)) []abci.ValidatorUpdate {
		ctx := a.NewUncachedContext(false, cmtproto.Header{Height: height, Time: at})
		if fn != nil {
			fn(ctx)
		}
		updates, err := a.StakingKeeper.EndBlocker(ctx)
		require.NoError(t, err)
		return updates
	}
	seat := func(ctx sdk.Context) {
		_, err := srv.AddValidator(ctx, &poatypes.MsgAddValidator{Authority: app.CouncilAuthority, ValidatorAddress: member.String(), Description: poatypes.Description{Moniker: "m"}, Pubkey: pk})
		require.NoError(t, err)
	}
	unseat := func(ctx sdk.Context) {
		_, err := srv.RemoveValidator(ctx, &poatypes.MsgRemoveValidator{Authority: app.CouncilAuthority, ValidatorAddress: member.String()})
		require.NoError(t, err)
	}
	t0 := time.Unix(100, 0).UTC()
	unbonding, err := a.StakingKeeper.UnbondingTime(a.NewUncachedContext(false, cmtproto.Header{Height: 2, Time: t0}))
	require.NoError(t, err)

	block(2, t0, seat)
	block(3, t0.Add(time.Second), unseat)
	block(4, t0.Add(unbonding+2*time.Second), nil) // matures: record deleted, bond back with the corporation
	ctx := a.NewUncachedContext(false, cmtproto.Header{Height: 5})
	_, err = a.StakingKeeper.GetValidator(ctx, valAddr)
	require.Error(t, err)
	require.True(t, a.BankKeeper.GetBalance(ctx, member, sdk.DefaultBondDenom).Amount.Equal(sdk.DefaultPowerReduction))

	updates := block(5, t0.Add(unbonding+3*time.Second), seat)
	require.Len(t, updates, 1)
	updates = block(6, t0.Add(unbonding+4*time.Second), unseat)
	require.Len(t, updates, 1)
	require.EqualValues(t, 0, updates[0].Power)
	v, err := a.StakingKeeper.GetValidator(a.NewUncachedContext(false, cmtproto.Header{Height: 7}), valAddr)
	require.NoError(t, err, "record deleted while CometBFT still holds the validator")
	require.True(t, v.IsUnbonding())
}

func TestAddValidator_NeedsOperatorBond(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	mutate(t, a, gs, banktypes.ModuleName, func(b *banktypes.GenesisState) {
		for i, bal := range b.Balances {
			if bal.Address == corps[1].policy.String() {
				b.Balances = append(b.Balances[:i], b.Balances[i+1:]...)
				break
			}
		}
	})
	require.NoError(t, initChain(t, a, gs))
	(&liveChain{app: a, now: time.Unix(0, 0).UTC()}).block(t)
	ctx := a.NewUncachedContext(false, cmtproto.Header{Height: 2, Time: time.Unix(2, 0).UTC()})
	supply0 := a.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom)
	pk, err := codectypes.NewAnyWithValue(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)
	_, err = poakeeper.NewMsgServerImpl(a.PoaKeeper).AddValidator(ctx, &poatypes.MsgAddValidator{
		Authority: app.CouncilAuthority, ValidatorAddress: corps[1].policy.String(), Description: poatypes.Description{Moniker: "m"}, Pubkey: pk,
	})
	require.ErrorContains(t, err, "insufficient funds")
	require.True(t, supply0.Equal(a.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom)))
}

func TestAuthorities_AreTheCouncil(t *testing.T) {
	a := newTestApp(t)
	council := sdk.MustAccAddressFromBech32(app.CouncilAuthority)
	require.Equal(t, council.Bytes(), a.XrKeeper.GetAuthority())
	require.Equal(t, app.CouncilAuthority, a.PoaKeeper.GetAuthority())
	require.Equal(t, app.CouncilAuthority, a.StakingKeeper.GetAuthority())
	require.Equal(t, app.CouncilAuthority, a.SlashingKeeper.GetAuthority())
	require.Equal(t, app.CouncilAuthority, a.BankKeeper.GetAuthority())
	require.Equal(t, app.CouncilAuthority, a.AccountKeeper.GetAuthority())
	require.Equal(t, app.CouncilAuthority, a.CrisisKeeper.GetAuthority())
	require.Equal(t, app.CouncilAuthority, a.ConsensusParamsKeeper.GetAuthority())
	require.Equal(t, app.CouncilAuthority, a.IBCKeeper.GetAuthority())
	require.Equal(t, app.CouncilAuthority, a.TransferKeeper.GetAuthority())
	require.Equal(t, app.CouncilAuthority, a.ICAHostKeeper.GetAuthority())
	require.Equal(t, app.CouncilAuthority, a.ICAControllerKeeper.GetAuthority())
}

func TestModuleAccounts_OnlyTransferMints(t *testing.T) {
	for name, perms := range app.GetMaccPerms() {
		for _, p := range perms {
			if p == authtypes.Minter {
				require.Equal(t, "transfer", name)
			}
		}
	}
	_, mint := app.GetMaccPerms()["mint"]
	require.False(t, mint)
	_, distr := app.GetMaccPerms()["distribution"]
	require.False(t, distr)
}

func TestSupply_StaysFixedAndFeesStayCollected(t *testing.T) {
	c, corps := seatedChain(t)
	ctx := c.app.NewContext(true)
	supply0 := c.app.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom)
	collector := authtypes.NewModuleAddress(authtypes.FeeCollectorName)
	fees0 := c.app.BankKeeper.GetBalance(ctx, collector, sdk.DefaultBondDenom)

	fee := sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1000))
	send := banktypes.NewMsgSend(corps[0].member(), corps[1].member(), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)))
	res := c.block(t, c.signedWithFee(t, corps[0].key, fee, send))
	require.Zero(t, res.TxResults[0].Code, res.TxResults[0].Log)
	for i := 0; i < 5; i++ {
		c.block(t)
	}

	ctx = c.app.NewContext(true)
	require.True(t, supply0.Equal(c.app.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom)))
	want := fees0.Add(fee[0])
	require.True(t, want.Equal(c.app.BankKeeper.GetBalance(ctx, collector, sdk.DefaultBondDenom)))
}

// liveChain drives the app block by block with signed transactions.
type liveChain struct {
	app    *app.App
	height int64
	now    time.Time
}

func (c *liveChain) block(t *testing.T, txs ...[]byte) *abci.ResponseFinalizeBlock {
	t.Helper()
	c.height++
	c.now = c.now.Add(time.Second)
	res, err := c.app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: c.height, Time: c.now, Txs: txs})
	require.NoError(t, err)
	_, err = c.app.Commit()
	require.NoError(t, err)
	return res
}

func (c *liveChain) signed(t *testing.T, priv cryptotypes.PrivKey, msgs ...sdk.Msg) []byte {
	t.Helper()
	return c.signedWithFee(t, priv, sdk.Coins{}, msgs...)
}

func (c *liveChain) signedWithFee(t *testing.T, priv cryptotypes.PrivKey, fee sdk.Coins, msgs ...sdk.Msg) []byte {
	t.Helper()
	acc := c.app.AccountKeeper.GetAccount(c.app.NewContext(true), sdk.AccAddress(priv.PubKey().Address()))
	require.NotNil(t, acc)
	tx, err := simtestutil.GenSignedMockTx(rand.New(rand.NewSource(1)), c.app.TxConfig(), msgs, fee, 2_000_000, "council-test",
		[]uint64{acc.GetAccountNumber()}, []uint64{acc.GetSequence()}, priv)
	require.NoError(t, err)
	bz, err := c.app.TxConfig().TxEncoder()(tx)
	require.NoError(t, err)
	return bz
}

// asCorp has a corporation execute msgs through its own threshold-1 group.
func (c *liveChain) asCorp(t *testing.T, co corp, msgs ...sdk.Msg) []byte {
	t.Helper()
	p := &group.MsgSubmitProposal{GroupPolicyAddress: co.policy.String(), Proposers: []string{co.member().String()}, Title: "t", Summary: "s", Exec: group.Exec_EXEC_TRY}
	require.NoError(t, p.SetMsgs(msgs))
	return c.signed(t, co.key, p)
}

func attr(ev abci.Event, key string) string {
	for _, a := range ev.Attributes {
		if a.Key == key {
			return strings.Trim(a.Value, "\"")
		}
	}
	return ""
}

// lastProposal returns the last proposal id submitted in a tx and whether every group execution in it succeeded.
func lastProposal(t *testing.T, res *abci.ExecTxResult) (uint64, bool) {
	t.Helper()
	var id uint64
	ok := true
	for _, ev := range res.Events {
		switch ev.Type {
		case "cosmos.group.v1.EventSubmitProposal":
			_, err := fmt.Sscan(attr(ev, "proposal_id"), &id)
			require.NoError(t, err)
		case "cosmos.group.v1.EventExec":
			if attr(ev, "result") != group.PROPOSAL_EXECUTOR_RESULT_SUCCESS.String() {
				ok = false
			}
		}
	}
	return id, ok
}

// councilVote submits msgs to the council from corps[0] and has the given corporations vote yes; returns the exec result.
func (c *liveChain) councilVote(t *testing.T, corps []corp, voters []int, msgs ...sdk.Msg) *abci.ExecTxResult {
	t.Helper()
	submit := &group.MsgSubmitProposal{GroupPolicyAddress: app.CouncilAuthority, Proposers: []string{corps[0].policy.String()}, Title: "t", Summary: "s"}
	require.NoError(t, submit.SetMsgs(msgs))
	res := c.block(t, c.asCorp(t, corps[0], submit))
	require.Zero(t, res.TxResults[0].Code, res.TxResults[0].Log)
	id, ok := lastProposal(t, res.TxResults[0])
	require.True(t, ok)

	var votes [][]byte
	for _, i := range voters {
		votes = append(votes, c.asCorp(t, corps[i], &group.MsgVote{ProposalId: id, Voter: corps[i].policy.String(), Option: group.VOTE_OPTION_YES}))
	}
	res = c.block(t, votes...)
	for _, r := range res.TxResults {
		require.Zero(t, r.Code, r.Log)
		_, ok := lastProposal(t, r)
		require.True(t, ok)
	}
	res = c.block(t, c.signed(t, corps[0].key, &group.MsgExec{ProposalId: id, Executor: corps[0].member().String()}))
	require.Zero(t, res.TxResults[0].Code, res.TxResults[0].Log)
	return res.TxResults[0]
}

// seatedChain starts a chain with a council of three corporations and one genesis validator.
func seatedChain(t *testing.T) (*liveChain, []corp) {
	t.Helper()
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	require.NoError(t, initChain(t, a, gs))
	c := &liveChain{app: a, now: time.Unix(0, 0).UTC()}
	c.block(t)
	return c, corps
}

func TestAnte_RejectsDelegationTx(t *testing.T) {
	c, corps := seatedChain(t)
	valoper := sdk.ValAddress(corps[0].policy).String()
	delegate := &stakingtypes.MsgDelegate{DelegatorAddress: corps[1].member().String(), ValidatorAddress: valoper, Amount: sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)}

	res := c.block(t, c.signed(t, corps[1].key, delegate))
	require.Equal(t, poatypes.ErrMsgNotAllowed.ABCICode(), res.TxResults[0].Code, res.TxResults[0].Log)
	require.Equal(t, poatypes.ModuleName, res.TxResults[0].Codespace)

	proposal := &group.MsgSubmitProposal{GroupPolicyAddress: app.CouncilAuthority, Proposers: []string{corps[1].policy.String()}, Title: "t", Summary: "s"}
	require.NoError(t, proposal.SetMsgs([]sdk.Msg{delegate}))
	res = c.block(t, c.asCorp(t, corps[1], proposal))
	require.Equal(t, poatypes.ErrMsgNotAllowed.ABCICode(), res.TxResults[0].Code, res.TxResults[0].Log)
	require.Equal(t, poatypes.ModuleName, res.TxResults[0].Codespace)

	send := banktypes.NewMsgSend(corps[1].member(), corps[0].member(), sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1)))
	res = c.block(t, c.signed(t, corps[1].key, send))
	require.Zero(t, res.TxResults[0].Code, res.TxResults[0].Log)
}

func TestAnte_RefusesCouncilLeave(t *testing.T) {
	c, corps := seatedChain(t)
	res := c.block(t, c.asCorp(t, corps[2], &group.MsgLeaveGroup{Address: corps[2].policy.String(), GroupId: 1}))
	require.Equal(t, poatypes.ErrCouncilIntegrity.ABCICode(), res.TxResults[0].Code, res.TxResults[0].Log)
}

func TestProposal_CorporationsSeatValidator(t *testing.T) {
	c, corps := seatedChain(t)
	supply0 := c.app.BankKeeper.GetSupply(c.app.NewContext(true), sdk.DefaultBondDenom)
	consPk, err := codectypes.NewAnyWithValue(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)

	exec := c.councilVote(t, corps, []int{0, 1}, &poatypes.MsgAddValidator{
		Authority: app.CouncilAuthority, ValidatorAddress: corps[1].policy.String(), Description: poatypes.Description{Moniker: "m1"}, Pubkey: consPk,
	})
	_, ok := lastProposal(t, exec)
	require.True(t, ok)

	ctx := c.app.NewContext(true)
	v, err := c.app.StakingKeeper.GetValidator(ctx, sdk.ValAddress(corps[1].policy))
	require.NoError(t, err)
	require.True(t, v.IsBonded())
	require.True(t, v.Tokens.Equal(sdk.DefaultPowerReduction))
	require.True(t, supply0.Equal(c.app.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom)))
}

func TestProposal_CouncilAdmitsOnlyCorporations(t *testing.T) {
	c, corps := seatedChain(t)
	key := corps[1].member().String()
	submit := &group.MsgSubmitProposal{GroupPolicyAddress: app.CouncilAuthority, Proposers: []string{corps[0].policy.String()}, Title: "t", Summary: "s"}
	require.NoError(t, submit.SetMsgs([]sdk.Msg{&group.MsgUpdateGroupMembers{
		Admin: app.CouncilAuthority, GroupId: 1, MemberUpdates: []group.MemberRequest{{Address: key, Weight: "1"}},
	}}))
	res := c.block(t, c.asCorp(t, corps[0], submit))
	require.Equal(t, poatypes.ErrNotACorporation.ABCICode(), res.TxResults[0].Code, res.TxResults[0].Log)
}

func TestProposal_CouncilAdmitsCorporation(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 4)
	dropFromCouncil(t, a, gs, corps[3].policy.String())
	require.NoError(t, initChain(t, a, gs))
	c := &liveChain{app: a, now: time.Unix(0, 0).UTC()}
	c.block(t)

	exec := c.councilVote(t, corps, []int{0, 1}, &group.MsgUpdateGroupMembers{
		Admin: app.CouncilAuthority, GroupId: 1, MemberUpdates: []group.MemberRequest{{Address: corps[3].policy.String(), Weight: "1"}},
	})
	_, ok := lastProposal(t, exec)
	require.True(t, ok)
	member, err := a.PoaKeeper.IsCouncilMember(a.NewContext(true), corps[3].policy.String())
	require.NoError(t, err)
	require.True(t, member)
}

func TestSetGenesisCouncil_RejectsTwoCorporations(t *testing.T) {
	a := newTestApp(t)
	gs := a.DefaultGenesis()
	spec := app.GenesisCouncil{Percentage: poatypes.CouncilPercentage, VotingPeriod: time.Hour, Corporations: []app.GenesisCorporation{
		{Did: "did:example:a", Language: "en", Members: []string{sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()}},
		{Did: "did:example:b", Language: "en", Members: []string{sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()}},
	}}
	require.ErrorContains(t, app.SetGenesisCouncil(a.AppCodec(), gs, spec, time.Unix(0, 0).UTC()), "at least 3")
}

func TestSetGenesisCouncil_FundsCorporationsWithSupply(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	mutate(t, a, gs, banktypes.ModuleName, func(b *banktypes.GenesisState) {
		var total sdk.Coins
		for _, bal := range b.Balances {
			total = total.Add(bal.Coins...)
		}
		b.Supply = total
	})
	spec := app.GenesisCouncil{Percentage: poatypes.CouncilPercentage, VotingPeriod: time.Hour}
	fresh := a.DefaultGenesis()
	fresh[banktypes.ModuleName] = gs[banktypes.ModuleName]
	extra := sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 7))
	for i := 0; i < 3; i++ {
		spec.Corporations = append(spec.Corporations, app.GenesisCorporation{
			Did: fmt.Sprintf("did:example:other%d", i), Language: "en", Members: []string{corps[i].member().String()}, Funds: extra,
		})
	}
	require.NoError(t, app.SetGenesisCouncil(a.AppCodec(), fresh, spec, time.Unix(0, 0).UTC()))
	var bank banktypes.GenesisState
	a.AppCodec().MustUnmarshalJSON(fresh[banktypes.ModuleName], &bank)
	require.NoError(t, bank.Validate())
	for _, b := range bank.Balances {
		if b.Address == corps[0].policy.String() {
			require.True(t, b.Coins.AmountOf(sdk.DefaultBondDenom).Equal(sdk.DefaultPowerReduction.AddRaw(7)))
		}
	}
}

func TestSetGenesisCouncil_CorporationWindows(t *testing.T) {
	a := newTestApp(t)
	gs := a.DefaultGenesis()
	spec := app.GenesisCouncil{Percentage: poatypes.CouncilPercentage, VotingPeriod: time.Hour, MinExecutionPeriod: 48 * time.Hour}
	for i := 0; i < 3; i++ {
		spec.Corporations = append(spec.Corporations, app.GenesisCorporation{
			Did: fmt.Sprintf("did:example:corp%d", i), Language: "en", Members: []string{sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()},
		})
	}
	spec.Corporations[0].VotingPeriod = 2 * time.Hour
	spec.Corporations[0].MinExecutionPeriod = 30 * time.Minute
	require.NoError(t, app.SetGenesisCouncil(a.AppCodec(), gs, spec, time.Unix(0, 0).UTC()))

	var gg group.GenesisState
	a.AppCodec().MustUnmarshalJSON(gs[group.ModuleName], &gg)
	windows := map[string][2]time.Duration{}
	for _, p := range gg.GroupPolicies {
		dp, err := p.GetDecisionPolicy()
		require.NoError(t, err)
		windows[p.Address] = [2]time.Duration{dp.GetVotingPeriod(), dp.GetMinExecutionPeriod()}
	}
	require.Equal(t, [2]time.Duration{2 * time.Hour, 30 * time.Minute}, windows[poatypes.GroupPolicyAddress(2).String()])
	require.Equal(t, [2]time.Duration{time.Hour, 0}, windows[poatypes.GroupPolicyAddress(3).String()])
	require.Equal(t, [2]time.Duration{time.Hour, 48 * time.Hour}, windows[app.CouncilAuthority])
}

func TestSetGenesisCouncil_AddsBondToNotBondedPool(t *testing.T) {
	a := newTestApp(t)
	gs, corps := councilGenesis(t, a, 3)
	var bank banktypes.GenesisState
	a.AppCodec().MustUnmarshalJSON(gs[banktypes.ModuleName], &bank)
	pool := authtypes.NewModuleAddress(stakingtypes.NotBondedPoolName).String()
	found := false
	for _, b := range bank.Balances {
		if b.Address == pool {
			found = true
			require.True(t, b.Coins.AmountOf(sdk.DefaultBondDenom).Equal(sdk.DefaultPowerReduction))
		}
	}
	require.True(t, found)

	var staking stakingtypes.GenesisState
	a.AppCodec().MustUnmarshalJSON(gs[stakingtypes.ModuleName], &staking)
	require.Len(t, staking.Validators, 1)
	require.Equal(t, sdk.ValAddress(corps[0].policy).String(), staking.Validators[0].OperatorAddress)
	require.Equal(t, math.LegacyNewDecFromInt(sdk.DefaultPowerReduction), staking.Delegations[0].Shares)
}
