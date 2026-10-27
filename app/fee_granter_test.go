package app

import (
	goruntime "runtime"
	"testing"
	"time"

	"cosmossdk.io/log/v2"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsign "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/stretchr/testify/require"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	pptypes "github.com/verana-labs/verana-node/x/pp/types"
)

const feeGranterChainID = "verana-test-1"

type feeGranterEnv struct {
	app    *App
	valSet *cmttypes.ValidatorSet
	now    time.Time
	height int64

	op, exec    cryptotypes.PrivKey
	corp, other sdk.AccAddress
	// feePID has with_feegrant = true, noFeePID has with_feegrant = false.
	feePID, noFeePID uint64
}

// newFeeGranterEnv starts a one-validator app and seeds a corporation with two
// HOLDER participants sharing one vs_operator. Only feePID enables corp-paid fees.
func newFeeGranterEnv(t *testing.T) *feeGranterEnv {
	t.Helper()
	opts := simtestutil.AppOptionsMap{flags.FlagHome: t.TempDir(), server.FlagInvCheckPeriod: uint(0)}
	a, err := New(log.NewNopLogger(), dbm.NewMemDB(), true, opts, baseapp.SetChainID(feeGranterChainID))
	require.NoError(t, err)

	valSet, err := simtestutil.CreateRandomValidatorSet()
	require.NoError(t, err)

	e := &feeGranterEnv{
		app: a, valSet: valSet, now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		op: secp256k1.GenPrivKey(), exec: secp256k1.GenPrivKey(),
		corp:  sdk.AccAddress([]byte("corp_policy_address_")),
		other: sdk.AccAddress([]byte("other_fee_granter___")),
	}
	coins := sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 1e12), sdk.NewInt64Coin("uvna", 1e12))
	var accs []authtypes.GenesisAccount
	var bals []banktypes.Balance
	for i, addr := range []sdk.AccAddress{e.opAddr(), e.execAddr(), e.corp, e.other} {
		accs = append(accs, authtypes.NewBaseAccount(addr, nil, uint64(i), 0))
		bals = append(bals, banktypes.Balance{Address: addr.String(), Coins: coins})
	}
	gen, err := simtestutil.GenesisStateWithValSet(a.AppCodec(), a.DefaultGenesis(), valSet, accs, bals...)
	require.NoError(t, err)
	state, err := cmtjson.MarshalIndent(gen, "", " ")
	require.NoError(t, err)

	params := *simtestutil.DefaultConsensusParams
	params.Block = &cmtproto.BlockParams{MaxBytes: 22020096, MaxGas: -1}
	params.Authority = nil
	_, err = a.InitChain(&abci.RequestInitChain{
		ChainId: feeGranterChainID, Time: e.now, ConsensusParams: &params, AppStateBytes: state,
	})
	require.NoError(t, err)
	e.finalize(t)

	ctx := e.nextBlockCtx()
	corp := cotypes.Corporation{Id: 1, PolicyAddress: e.corp.String(), Did: "did:example:corp"}
	require.NoError(t, a.CoKeeper.Corporation.Set(ctx, corp.Id, corp))
	require.NoError(t, a.CoKeeper.CorporationByPolicyAddr.Set(ctx, corp.PolicyAddress, corp.Id))
	from := e.now.Add(-time.Hour)
	for _, p := range []struct {
		pid     *uint64
		did     string
		withFee bool
	}{{&e.feePID, "did:example:fee", true}, {&e.noFeePID, "did:example:nofee", false}} {
		id, err := a.ParticipantKeeper.CreateParticipant(ctx, pptypes.Participant{
			SchemaId: 1, Role: pptypes.ParticipantRole_HOLDER, Did: p.did,
			CorporationId: corp.Id, VsOperator: e.opAddr().String(), EffectiveFrom: &from,
		})
		require.NoError(t, err)
		*p.pid = id
		rec := detypes.ParticipantAuthorizationRecord{
			ParticipantId: id, MsgTypes: []string{pptypes.MsgTriggerResolverTypeURL}, WithFeegrant: p.withFee,
		}
		if p.withFee {
			rec.FeeSpendLimit = sdk.NewCoins(sdk.NewInt64Coin("uvna", 1e9))
		}
		require.NoError(t, a.DeKeeper.GrantVSOperatorAuthorization(ctx, corp.Id, e.opAddr().String(), rec))
	}
	// A plain allowance from corp to exec (for MsgExec) and from other to op.
	require.NoError(t, a.FeeGrantKeeper.GrantAllowance(ctx, e.corp, e.execAddr(), &feegrant.BasicAllowance{}))
	require.NoError(t, a.FeeGrantKeeper.GrantAllowance(ctx, e.other, e.opAddr(), &feegrant.BasicAllowance{}))
	require.NoError(t, a.AuthzKeeper.SaveGrant(ctx, e.execAddr(), e.opAddr(),
		authz.NewGenericAuthorization(pptypes.MsgTriggerResolverTypeURL), nil))
	e.finalize(t)
	return e
}

func (e *feeGranterEnv) opAddr() sdk.AccAddress   { return sdk.AccAddress(e.op.PubKey().Address()) }
func (e *feeGranterEnv) execAddr() sdk.AccAddress { return sdk.AccAddress(e.exec.PubKey().Address()) }

func (e *feeGranterEnv) nextBlockCtx() sdk.Context {
	return e.app.NewNextBlockContext(cmtproto.Header{
		ChainID: feeGranterChainID, Height: e.height + 1, Time: e.now,
	})
}

// finalize commits one block with txs and returns their results.
func (e *feeGranterEnv) finalize(t *testing.T, txs ...[]byte) []*abci.ExecTxResult {
	t.Helper()
	e.height++
	res, err := e.app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: e.height, Time: e.now, Txs: txs, NextValidatorsHash: e.valSet.Hash(),
	})
	require.NoError(t, err)
	_, err = e.app.Commit()
	require.NoError(t, err)
	return res.TxResults
}

func (e *feeGranterEnv) trigger(pid uint64) *pptypes.MsgTriggerResolver {
	return &pptypes.MsgTriggerResolver{Corporation: e.corp.String(), Operator: e.opAddr().String(), Id: pid}
}

// signedTx signs msgs with signer, paying 1000uvna through granter when set.
func (e *feeGranterEnv) signedTx(t *testing.T, signer cryptotypes.PrivKey, granter sdk.AccAddress, msgs ...sdk.Msg) []byte {
	t.Helper()
	txCfg := e.app.TxConfig()
	acc := e.app.AccountKeeper.GetAccount(e.app.NewContext(true), sdk.AccAddress(signer.PubKey().Address()))
	signMode, err := authsign.APISignModeToInternal(txCfg.SignModeHandler().DefaultMode())
	require.NoError(t, err)

	b := txCfg.NewTxBuilder()
	require.NoError(t, b.SetMsgs(msgs...))
	b.SetGasLimit(1 << 40)
	b.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin("uvna", 1000)))
	if granter != nil {
		b.SetFeeGranter(granter)
	}
	sig := signing.SignatureV2{
		PubKey:   signer.PubKey(),
		Data:     &signing.SingleSignatureData{SignMode: signMode},
		Sequence: acc.GetSequence(),
	}
	require.NoError(t, b.SetSignatures(sig))
	signBytes, err := authsign.GetSignBytesAdapter(t.Context(), txCfg.SignModeHandler(), signMode, authsign.SignerData{
		Address: acc.GetAddress().String(), ChainID: feeGranterChainID,
		AccountNumber: acc.GetAccountNumber(), Sequence: acc.GetSequence(), PubKey: signer.PubKey(),
	}, b.GetTx())
	require.NoError(t, err)
	sig.Data.(*signing.SingleSignatureData).Signature, err = signer.Sign(signBytes)
	require.NoError(t, err)
	require.NoError(t, b.SetSignatures(sig))

	bz, err := txCfg.TxEncoder()(b.GetTx())
	require.NoError(t, err)
	return bz
}

func (e *feeGranterEnv) uvna(addr sdk.AccAddress) int64 {
	return e.app.BankKeeper.GetBalance(e.app.NewContext(true), addr, "uvna").Amount.Int64()
}

// The work per MsgTriggerResolver must not grow with the size of the tx that
// carries it.
func TestTriggerResolverCostPerMsgIndependentOfTxSize(t *testing.T) {
	e := newFeeGranterEnv(t)
	perMsg := func(n int) float64 {
		msgs := make([]sdk.Msg, n)
		for i := range msgs {
			msgs[i] = e.trigger(e.noFeePID)
		}
		bz := e.signedTx(t, e.op, nil, msgs...)
		var before, after goruntime.MemStats
		goruntime.GC()
		goruntime.ReadMemStats(&before)
		_, _, err := e.app.Simulate(bz)
		goruntime.ReadMemStats(&after)
		require.NoError(t, err)
		return float64(after.TotalAlloc-before.TotalAlloc) / float64(n)
	}
	small, large := perMsg(200), perMsg(800)
	t.Logf("bytes allocated per msg: 200 msgs %.0f, 800 msgs %.0f", small, large)
	require.Less(t, large, 2*small)
}

// AUTHZ-CHECK-4 through the real ante chain: the fee granter of the tx decides
// whether the record must have with_feegrant.
func TestVSOperatorFeeGrantCheckUsesTxFeeGranter(t *testing.T) {
	e := newFeeGranterEnv(t)
	const notAuthorized = "fee grant not authorized"

	cases := []struct {
		name    string
		signer  cryptotypes.PrivKey
		granter sdk.AccAddress
		msgs    []sdk.Msg
		payer   sdk.AccAddress
		errLog  string
	}{
		{"corp pays, with_feegrant", e.op, e.corp, []sdk.Msg{e.trigger(e.feePID)}, e.corp, ""},
		{"corp pays, no with_feegrant", e.op, e.corp, []sdk.Msg{e.trigger(e.noFeePID)}, e.corp, notAuthorized},
		{"corp pays, second msg lacks with_feegrant", e.op, e.corp, []sdk.Msg{e.trigger(e.feePID), e.trigger(e.noFeePID)}, e.corp, notAuthorized},
		{"operator pays", e.op, nil, []sdk.Msg{e.trigger(e.noFeePID)}, e.opAddr(), ""},
		{"other granter pays", e.op, e.other, []sdk.Msg{e.trigger(e.noFeePID)}, e.other, ""},
		{"MsgExec, corp pays, with_feegrant", e.exec, e.corp, []sdk.Msg{execMsg(e, e.trigger(e.feePID))}, e.corp, ""},
		{"MsgExec, corp pays, no with_feegrant", e.exec, e.corp, []sdk.Msg{execMsg(e, e.trigger(e.noFeePID))}, e.corp, notAuthorized},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bz := e.signedTx(t, tc.signer, tc.granter, tc.msgs...)

			_, _, simErr := e.app.Simulate(bz)
			before := e.uvna(tc.payer)
			res := e.finalize(t, bz)[0]
			require.Equal(t, before-1000, e.uvna(tc.payer), "fee payer")
			if tc.errLog == "" {
				require.NoError(t, simErr)
				require.Zero(t, res.Code, res.Log)
				return
			}
			require.ErrorContains(t, simErr, tc.errLog)
			require.NotZero(t, res.Code)
			require.Contains(t, res.Log, tc.errLog)
		})
	}
}

func execMsg(e *feeGranterEnv, msgs ...sdk.Msg) sdk.Msg {
	m := authz.NewMsgExec(e.execAddr(), msgs)
	return &m
}
