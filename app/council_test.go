package app_test

import (
	"encoding/json"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/group"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	icagenesistypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/genesis/types"
	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/app"
	poakeeper "github.com/verana-labs/verana-node/x/poa/keeper"
	poatypes "github.com/verana-labs/verana-node/x/poa/types"
)

func newTestApp(t *testing.T) *app.App {
	t.Helper()
	a, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true,
		simtestutil.NewAppOptionsWithFlagHome(t.TempDir()), baseapp.SetChainID("council-test"))
	require.NoError(t, err)
	return a
}

// genesisWith returns a genesis with one validator and the given council members;
// nil members means no council at all. The validator operator address is returned.
func genesisWith(t *testing.T, a *app.App, members func(operator string) []string) map[string]json.RawMessage {
	t.Helper()
	pk := ed25519.GenPrivKey().PubKey()
	cmtPk, err := cryptocodec.ToCmtPubKeyInterface(pk)
	require.NoError(t, err)
	val := cmttypes.NewValidator(cmtPk, 1)
	operator := sdk.AccAddress(val.Address)
	acc := authtypes.NewBaseAccount(operator, nil, 0, 0)
	bal := banktypes.Balance{Address: operator.String(), Coins: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100_000_000)))}
	gs, err := simtestutil.GenesisStateWithValSet(a.AppCodec(), a.DefaultGenesis(),
		cmttypes.NewValidatorSet([]*cmttypes.Validator{val}), []authtypes.GenesisAccount{acc}, bal)
	require.NoError(t, err)
	var ica icagenesistypes.GenesisState
	a.AppCodec().MustUnmarshalJSON(gs[icatypes.ModuleName], &ica)
	ica.HostGenesisState.Params.AllowMessages = []string{"/cosmos.bank.v1beta1.MsgSend"}
	gs[icatypes.ModuleName] = a.AppCodec().MustMarshalJSON(&ica)
	if members != nil {
		cg, err := poatypes.CouncilGenesis(app.CouncilAuthority, members(operator.String()), "test council",
			poatypes.CouncilPercentage, time.Hour, 0, time.Unix(0, 0).UTC())
		require.NoError(t, err)
		gs[group.ModuleName] = a.AppCodec().MustMarshalJSON(cg)
	}
	return gs
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

func TestInitChain_RequiresCouncil(t *testing.T) {
	a := newTestApp(t)
	err := initChain(t, a, genesisWith(t, a, nil))
	require.ErrorContains(t, err, "no council group policy")
}

func TestInitChain_RejectsValidatorOutsideCouncil(t *testing.T) {
	a := newTestApp(t)
	other := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()).String()
	err := initChain(t, a, genesisWith(t, a, func(string) []string { return []string{other} }))
	require.ErrorContains(t, err, "not a council member")
}

func TestInitChain_AcceptsCouncilValidator(t *testing.T) {
	a := newTestApp(t)
	require.NoError(t, initChain(t, a, genesisWith(t, a, func(op string) []string { return []string{op} })))
}

func TestInitChain_RejectsOpenICAHost(t *testing.T) {
	a := newTestApp(t)
	gs := genesisWith(t, a, func(op string) []string { return []string{op} })
	var ica icagenesistypes.GenesisState
	a.AppCodec().MustUnmarshalJSON(gs[icatypes.ModuleName], &ica)
	ica.HostGenesisState.Params.AllowMessages = []string{"*"}
	gs[icatypes.ModuleName] = a.AppCodec().MustMarshalJSON(&ica)
	require.ErrorContains(t, initChain(t, a, gs), "ica host allow_messages")
}

func mutateCouncil(t *testing.T, a *app.App, gs map[string]json.RawMessage, fn func(*group.GenesisState)) {
	t.Helper()
	var cg group.GenesisState
	a.AppCodec().MustUnmarshalJSON(gs[group.ModuleName], &cg)
	fn(&cg)
	gs[group.ModuleName] = a.AppCodec().MustMarshalJSON(&cg)
}

func TestInitChain_RejectsForeignPolicyAdmin(t *testing.T) {
	a := newTestApp(t)
	gs := genesisWith(t, a, func(op string) []string { return []string{op} })
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) { cg.GroupPolicies[0].Admin = cg.GroupMembers[0].Member.Address })
	require.ErrorContains(t, initChain(t, a, gs), "policy admin must be the policy")
}

func TestInitChain_RejectsWeightedMember(t *testing.T) {
	a := newTestApp(t)
	gs := genesisWith(t, a, func(op string) []string { return []string{op} })
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) { cg.GroupMembers[0].Member.Weight = "2"; cg.Groups[0].TotalWeight = "2" })
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
	gs := genesisWith(t, a, func(op string) []string { return []string{op} })
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) {
		prebake(t, cg, group.PROPOSAL_STATUS_SUBMITTED, &stakingtypes.MsgDelegate{DelegatorAddress: app.CouncilAuthority, ValidatorAddress: "v", Amount: sdk.NewInt64Coin("stake", 1)})
	})
	require.ErrorContains(t, initChain(t, a, gs), "not allowed")
}

func TestInitChain_KeepsHistoricalProposals(t *testing.T) {
	a := newTestApp(t)
	gs := genesisWith(t, a, func(op string) []string { return []string{op} })
	mutateCouncil(t, a, gs, func(cg *group.GenesisState) {
		prebake(t, cg, group.PROPOSAL_STATUS_REJECTED, &stakingtypes.MsgDelegate{DelegatorAddress: app.CouncilAuthority, ValidatorAddress: "v", Amount: sdk.NewInt64Coin("stake", 1)})
	})
	require.NoError(t, initChain(t, a, gs))
}

func TestSeatAndUnseat_ThroughStaking(t *testing.T) {
	a := newTestApp(t)
	member := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()).String()
	require.NoError(t, initChain(t, a, genesisWith(t, a, func(op string) []string { return []string{op, member} })))
	_, err := a.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: time.Unix(1, 0).UTC()})
	require.NoError(t, err)
	_, err = a.Commit()
	require.NoError(t, err)

	ctx := a.NewUncachedContext(false, cmtproto.Header{Height: 2, Time: time.Unix(2, 0).UTC()})
	supply0 := a.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom)
	pk, err := codectypes.NewAnyWithValue(ed25519.GenPrivKey().PubKey())
	require.NoError(t, err)
	srv := poakeeper.NewMsgServerImpl(a.PoaKeeper)

	_, err = srv.AddValidator(ctx, &poatypes.MsgAddValidator{Authority: app.CouncilAuthority, ValidatorAddress: member, Description: poatypes.Description{Moniker: "m"}, Pubkey: pk})
	require.NoError(t, err)
	updates, err := a.StakingKeeper.EndBlocker(ctx)
	require.NoError(t, err)
	require.Len(t, updates, 1)
	require.EqualValues(t, 1, updates[0].Power)

	_, err = srv.RemoveValidator(ctx, &poatypes.MsgRemoveValidator{Authority: app.CouncilAuthority, ValidatorAddress: member})
	require.NoError(t, err)
	updates, err = a.StakingKeeper.EndBlocker(ctx)
	require.NoError(t, err)
	require.Len(t, updates, 1)
	require.EqualValues(t, 0, updates[0].Power)
	require.True(t, supply0.Equal(a.BankKeeper.GetSupply(ctx, sdk.DefaultBondDenom)))

	// the record must survive until unbonding_time so distribution can still resolve its last votes
	memberAcc := sdk.MustAccAddressFromBech32(member)
	v, err := a.StakingKeeper.GetValidator(ctx, sdk.ValAddress(memberAcc))
	require.NoError(t, err)
	require.True(t, v.IsUnbonding(), v.Status.String())
}

func TestAuthorities_AreTheCouncil(t *testing.T) {
	a := newTestApp(t)
	council := sdk.MustAccAddressFromBech32(app.CouncilAuthority)
	require.Equal(t, "verana1eu3mt98mqvxhmrtjyg209xuld22ssahzt40ewz", app.FrozenAuthority)
	require.Equal(t, council.Bytes(), a.XrKeeper.GetAuthority())
	require.Equal(t, app.CouncilAuthority, a.PoaKeeper.GetAuthority())
	require.Equal(t, app.FrozenAuthority, a.StakingKeeper.GetAuthority())
	require.NotEqual(t, app.CouncilAuthority, app.FrozenAuthority)
}

func TestReseatAfterUnbonding_KeepsRecordUntilMature(t *testing.T) {
	a := newTestApp(t)
	member := sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()).String()
	require.NoError(t, initChain(t, a, genesisWith(t, a, func(op string) []string { return []string{op, member} })))
	_, err := a.FinalizeBlock(&abci.RequestFinalizeBlock{Height: 1, Time: time.Unix(1, 0).UTC()})
	require.NoError(t, err)
	_, err = a.Commit()
	require.NoError(t, err)

	valAddr := sdk.ValAddress(sdk.MustAccAddressFromBech32(member))
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
		_, err := srv.AddValidator(ctx, &poatypes.MsgAddValidator{Authority: app.CouncilAuthority, ValidatorAddress: member, Description: poatypes.Description{Moniker: "m"}, Pubkey: pk})
		require.NoError(t, err)
	}
	unseat := func(ctx sdk.Context) {
		_, err := srv.RemoveValidator(ctx, &poatypes.MsgRemoveValidator{Authority: app.CouncilAuthority, ValidatorAddress: member})
		require.NoError(t, err)
	}
	t0 := time.Unix(100, 0).UTC()
	unbonding, err := a.StakingKeeper.UnbondingTime(a.NewUncachedContext(false, cmtproto.Header{Height: 2, Time: t0}))
	require.NoError(t, err)

	block(2, t0, seat)
	block(3, t0.Add(time.Second), unseat)
	block(4, t0.Add(unbonding+2*time.Second), nil) // matures and deletes the record
	_, err = a.StakingKeeper.GetValidator(a.NewUncachedContext(false, cmtproto.Header{Height: 5}), valAddr)
	require.Error(t, err)

	updates := block(5, t0.Add(unbonding+3*time.Second), seat)
	require.Len(t, updates, 1)
	updates = block(6, t0.Add(unbonding+4*time.Second), unseat)
	require.Len(t, updates, 1)
	require.EqualValues(t, 0, updates[0].Power)
	v, err := a.StakingKeeper.GetValidator(a.NewUncachedContext(false, cmtproto.Header{Height: 7}), valAddr)
	require.NoError(t, err, "record deleted while CometBFT still holds the validator")
	require.True(t, v.IsUnbonding())
}
