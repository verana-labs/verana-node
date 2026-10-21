package app

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/log/v2"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	dekeeper "github.com/verana-labs/verana-node/x/de/keeper"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	ppkeeper "github.com/verana-labs/verana-node/x/pp/keeper"
	pptypes "github.com/verana-labs/verana-node/x/pp/types"
	tdtypes "github.com/verana-labs/verana-node/x/td/types"
)

func feegrantTestApp(t *testing.T) (*App, sdk.Context) {
	t.Helper()
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = DefaultNodeHome
	appOptions[server.FlagInvCheckPeriod] = uint(0)
	a, err := New(log.NewNopLogger(), dbm.NewMemDB(), true, appOptions)
	require.NoError(t, err)
	ctx := a.NewContextLegacy(true, cmtproto.Header{Height: 10, Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}).WithIsCheckTx(false)
	require.NoError(t, a.DeKeeper.Params.Set(ctx, detypes.DefaultParams()))
	return a, ctx
}

func testAllowances(t *testing.T, now time.Time) map[string]feegrant.FeeAllowanceI {
	t.Helper()
	limit := sdk.NewCoins(sdk.NewInt64Coin("uvna", 10))
	exp := now.Add(time.Hour)
	inner := map[string]feegrant.FeeAllowanceI{
		"periodic":        &feegrant.PeriodicAllowance{Period: 24 * time.Hour, PeriodSpendLimit: limit, PeriodCanSpend: limit, PeriodReset: now.Add(24 * time.Hour)},
		"basic_expiring":  &feegrant.BasicAllowance{SpendLimit: limit, Expiration: &exp},
		"basic_unlimited": &feegrant.BasicAllowance{},
	}
	out := map[string]feegrant.FeeAllowanceI{}
	for name, a := range inner {
		allowed, err := feegrant.NewAllowedMsgAllowance(a, []string{pptypes.MsgCreateOrUpdateParticipantSessionTypeURL})
		require.NoError(t, err)
		out[name] = allowed
	}
	return out
}

func requireRevoked(t *testing.T, ctx sdk.Context, fk feegrantkeeper.Keeper, granter, grantee sdk.AccAddress, allowance feegrant.FeeAllowanceI) {
	t.Helper()
	_, err := fk.GetAllowance(ctx, granter, grantee)
	require.ErrorIs(t, err, collections.ErrNotFound)
	exp, err := allowance.ExpiresAt()
	require.NoError(t, err)
	if exp != nil {
		has, err := fk.FeeAllowanceQueue.Has(ctx, collections.Join3(*exp, grantee, granter))
		require.NoError(t, err)
		require.False(t, has)
	}
}

func TestDeFeegrantAdapter_RevokeSelfGrant(t *testing.T) {
	a, ctx := feegrantTestApp(t)
	p := sdk.AccAddress([]byte("corp_policy_________"))
	adapter := newDeFeegrantAdapter(a.FeeGrantKeeper)

	for name, allowance := range testAllowances(t, ctx.BlockTime()) {
		t.Run(name, func(t *testing.T) {
			c, _ := ctx.CacheContext()
			require.NoError(t, a.FeeGrantKeeper.GrantAllowance(c, p, p, allowance))

			c = c.WithEventManager(sdk.NewEventManager())
			require.NoError(t, adapter.RevokeAllowance(c, p, p))
			requireRevoked(t, c, a.FeeGrantKeeper, p, p, allowance)
			require.Equal(t, sdk.Events{sdk.NewEvent(feegrant.EventTypeRevokeFeeGrant,
				sdk.NewAttribute(feegrant.AttributeKeyGranter, p.String()),
				sdk.NewAttribute(feegrant.AttributeKeyGrantee, p.String()),
			)}, c.EventManager().Events())

			require.NoError(t, a.FeeGrantKeeper.GrantAllowance(c, p, p, allowance))
		})
	}
}

func TestDeFeegrantAdapter_RevokeMatchesMsgServer(t *testing.T) {
	a, ctx := feegrantTestApp(t)
	granter := sdk.AccAddress([]byte("corp_policy_________"))
	grantee := sdk.AccAddress([]byte("operator____________"))
	adapter := newDeFeegrantAdapter(a.FeeGrantKeeper)
	msgServer := feegrantkeeper.NewMsgServerImpl(a.FeeGrantKeeper)

	for name, allowance := range testAllowances(t, ctx.BlockTime()) {
		t.Run(name, func(t *testing.T) {
			base, _ := ctx.CacheContext()
			require.NoError(t, a.FeeGrantKeeper.GrantAllowance(base, granter, grantee, allowance))

			viaMsg, _ := base.CacheContext()
			viaMsg = viaMsg.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
			_, err := msgServer.RevokeAllowance(viaMsg, &feegrant.MsgRevokeAllowance{Granter: granter.String(), Grantee: grantee.String()})
			require.NoError(t, err)

			viaAdapter, _ := base.CacheContext()
			viaAdapter = viaAdapter.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithEventManager(sdk.NewEventManager())
			require.NoError(t, adapter.RevokeAllowance(viaAdapter, granter, grantee))

			require.Equal(t, viaMsg.GasMeter().GasConsumed(), viaAdapter.GasMeter().GasConsumed())
			require.Equal(t, viaMsg.EventManager().Events(), viaAdapter.EventManager().Events())
			requireRevoked(t, viaMsg, a.FeeGrantKeeper, granter, grantee, allowance)
			requireRevoked(t, viaAdapter, a.FeeGrantKeeper, granter, grantee, allowance)
		})
	}
}

func TestDeFeegrantAdapter_RevokeNotFound(t *testing.T) {
	a, ctx := feegrantTestApp(t)
	p := sdk.AccAddress([]byte("corp_policy_________"))
	o := sdk.AccAddress([]byte("operator____________"))
	adapter := newDeFeegrantAdapter(a.FeeGrantKeeper)

	require.ErrorIs(t, adapter.RevokeAllowance(ctx, p, o), collections.ErrNotFound)
	require.ErrorIs(t, adapter.RevokeAllowance(ctx, p, p), collections.ErrNotFound)
}

type selfGrantEnv struct {
	a        *App
	ctx      sdk.Context
	pE, pX   sdk.AccAddress
	opE, opX sdk.AccAddress
	rootID   uint64
	pps      pptypes.MsgServer
	des      detypes.MsgServer
}

// newSelfGrantEnv sets up an ecosystem corporation E with an OPEN-mode schema
// and a root participant, and a corporation X with an operator for MSG-14.
func newSelfGrantEnv(t *testing.T) selfGrantEnv {
	t.Helper()
	a, ctx := feegrantTestApp(t)
	e := selfGrantEnv{
		a: a, ctx: ctx,
		pE:  sdk.AccAddress([]byte("policy_ecosystem_E__")),
		pX:  sdk.AccAddress([]byte("policy_corp_X_______")),
		opE: sdk.AccAddress([]byte("operator_E__________")),
		opX: sdk.AccAddress([]byte("operator_X__________")),
		pps: ppkeeper.NewMsgServerImpl(a.ParticipantKeeper),
		des: dekeeper.NewMsgServerImpl(a.DeKeeper),
	}
	for id, addr := range map[uint64]sdk.AccAddress{1: e.pE, 2: e.pX} {
		require.NoError(t, a.CoKeeper.Corporation.Set(ctx, id, cotypes.Corporation{Id: id, PolicyAddress: addr.String(), Did: "did:example:corp" + string(rune('a'+id))}))
		require.NoError(t, a.CoKeeper.CorporationByPolicyAddr.Set(ctx, addr.String(), id))
	}
	require.NoError(t, a.CredentialschemaKeeper.CredentialSchema.Set(ctx, 1, cstypes.CredentialSchema{
		Id: 1, EcosystemId: 1, IssuerOnboardingMode: cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
		PricingAssetType: cstypes.PricingAssetType_COIN, PricingAsset: "uvna",
	}))
	from := ctx.BlockTime().Add(-time.Hour)
	rootID, err := a.ParticipantKeeper.CreateParticipant(ctx, pptypes.Participant{
		Role: pptypes.ParticipantRole_ECOSYSTEM, SchemaId: 1, CorporationId: 1, Did: "did:example:root",
		EffectiveFrom: &from, OpState: pptypes.OnboardingState_VALIDATED,
	})
	require.NoError(t, err)
	e.rootID = rootID

	_, err = e.des.GrantOperatorAuthorization(ctx, &detypes.MsgGrantOperatorAuthorization{
		Corporation: e.pX.String(), Operator: e.pX.String(), Grantee: e.opX.String(),
		MsgTypes: []string{"/verana.pp.v1.MsgSelfCreateParticipant", "/verana.pp.v1.MsgSetParticipantEffectiveUntil"},
	})
	require.NoError(t, err)
	_, err = e.des.GrantOperatorAuthorization(ctx, &detypes.MsgGrantOperatorAuthorization{
		Corporation: e.pE.String(), Operator: e.pE.String(), Grantee: e.opE.String(),
		MsgTypes: []string{"/verana.pp.v1.MsgRevokeParticipant"},
	})
	require.NoError(t, err)
	return e
}

// selfCreate creates an ISSUER of X whose vs_operator is X's own policy address.
func (e selfGrantEnv) selfCreate(t *testing.T, effectiveUntil *time.Time) uint64 {
	t.Helper()
	msg := &pptypes.MsgSelfCreateParticipant{
		Corporation: e.pX.String(), Operator: e.opX.String(), Role: pptypes.ParticipantRole_ISSUER,
		ValidatorParticipantId: e.rootID, Did: "did:example:issuerx", VsOperator: e.pX.String(),
		EffectiveUntil:               effectiveUntil,
		VsOperatorAuthzMsgTypes:      []string{pptypes.MsgCreateOrUpdateParticipantSessionTypeURL},
		VsOperatorAuthzWithFeegrant:  true,
		VsOperatorAuthzFeeSpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uvna", 1)),
	}
	require.NoError(t, msg.ValidateBasic())
	res, err := e.pps.SelfCreateParticipant(e.ctx, msg)
	require.NoError(t, err)
	_, err = e.a.FeeGrantKeeper.GetAllowance(e.ctx, e.pX, e.pX)
	require.NoError(t, err)
	return res.Id
}

func (e selfGrantEnv) requireNoSelfGrant(t *testing.T, ctx sdk.Context) {
	t.Helper()
	_, err := e.a.FeeGrantKeeper.GetAllowance(ctx, e.pX, e.pX)
	require.ErrorIs(t, err, collections.ErrNotFound)
	has, err := e.a.DeKeeper.FeeGrants.Has(ctx, collections.Join(uint64(2), e.pX.String()))
	require.NoError(t, err)
	require.False(t, has)
}

func TestSelfFeeGrant_RevokeParticipant(t *testing.T) {
	e := newSelfGrantEnv(t)
	id := e.selfCreate(t, nil)

	msg := &pptypes.MsgRevokeParticipant{Corporation: e.pE.String(), Operator: e.opE.String(), Id: id}
	_, err := e.a.MsgServiceRouter().Handler(msg)(e.ctx, msg)
	require.NoError(t, err)

	p, err := e.a.ParticipantKeeper.GetParticipantByID(e.ctx, id)
	require.NoError(t, err)
	require.NotNil(t, p.Revoked)
	e.requireNoSelfGrant(t, e.ctx)
	has, err := e.a.DeKeeper.VSOAByParticipant.Has(e.ctx, id)
	require.NoError(t, err)
	require.False(t, has)
}

func TestSelfFeeGrant_OperatorAuthorization(t *testing.T) {
	e := newSelfGrantEnv(t)
	X := e.pX.String()
	grant := func(withFeegrant bool) error {
		msg := &detypes.MsgGrantOperatorAuthorization{Corporation: X, Operator: X, Grantee: X, MsgTypes: []string{"/verana.ec.v1.MsgCreateEcosystem"}, WithFeegrant: withFeegrant}
		if withFeegrant {
			msg.FeegrantSpendLimit = sdk.NewCoins(sdk.NewInt64Coin("uvna", 10))
		}
		_, err := e.des.GrantOperatorAuthorization(e.ctx, msg)
		return err
	}

	require.NoError(t, grant(true))
	require.NoError(t, grant(true))
	_, err := e.a.FeeGrantKeeper.GetAllowance(e.ctx, e.pX, e.pX)
	require.NoError(t, err)

	require.NoError(t, grant(false))
	e.requireNoSelfGrant(t, e.ctx)

	require.NoError(t, grant(true))
	_, err = e.des.RevokeOperatorAuthorization(e.ctx, &detypes.MsgRevokeOperatorAuthorization{Corporation: X, Operator: X, Grantee: X})
	require.NoError(t, err)
	e.requireNoSelfGrant(t, e.ctx)
}

func TestDeFeegrantAdapter_RevokeExpiredSelfGrant(t *testing.T) {
	a, ctx := feegrantTestApp(t)
	p := sdk.AccAddress([]byte("corp_policy_________"))
	adapter := newDeFeegrantAdapter(a.FeeGrantKeeper)

	for name, revokeAt := range map[string]time.Duration{"at_expiration": 0, "after_expiration": time.Hour} {
		t.Run(name, func(t *testing.T) {
			c, _ := ctx.CacheContext()
			exp := c.BlockTime()
			allowance := &feegrant.BasicAllowance{SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uvna", 10)), Expiration: &exp}
			require.NoError(t, a.FeeGrantKeeper.GrantAllowance(c, p, p, allowance))

			c = c.WithBlockTime(exp.Add(revokeAt))
			require.NoError(t, adapter.RevokeAllowance(c, p, p))
			requireRevoked(t, c, a.FeeGrantKeeper, p, p, allowance)
			require.NoError(t, a.FeeGrantKeeper.RemoveExpiredAllowances(c, 100))
		})
	}
}

func TestSelfFeeGrant_SetEffectiveUntil(t *testing.T) {
	e := newSelfGrantEnv(t)
	id := e.selfCreate(t, nil)

	until := e.ctx.BlockTime().Add(48 * time.Hour)
	_, err := e.pps.SetParticipantEffectiveUntil(e.ctx, &pptypes.MsgSetParticipantEffectiveUntil{Corporation: e.pX.String(), Operator: e.opX.String(), Id: id, EffectiveUntil: &until})
	require.NoError(t, err)

	_, err = e.a.FeeGrantKeeper.GetAllowance(e.ctx, e.pX, e.pX)
	require.NoError(t, err)
	has, err := e.a.DeKeeper.WindowEndQueue.Has(e.ctx, collections.Join(until, id))
	require.NoError(t, err)
	require.True(t, has)
}

func TestSelfFeeGrant_WindowEnd(t *testing.T) {
	e := newSelfGrantEnv(t)
	until := e.ctx.BlockTime().Add(time.Hour)
	e.selfCreate(t, &until)

	ctx := e.ctx.WithBlockTime(until.Add(time.Second))
	require.NoError(t, e.a.DeKeeper.EndBlock(ctx))
	e.requireNoSelfGrant(t, ctx)
}

func TestSelfFeeGrant_ValidatedEntry(t *testing.T) {
	e := newSelfGrantEnv(t)
	a, ctx := e.a, e.ctx
	require.NoError(t, a.TrustdepositKeeper.SetParams(ctx, tdtypes.DefaultParams()))
	require.NoError(t, a.CredentialschemaKeeper.CredentialSchema.Set(ctx, 2, cstypes.CredentialSchema{
		Id: 2, EcosystemId: 1, IssuerOnboardingMode: cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_ECOSYSTEM_ONBOARDING_PROCESS,
		IssuerValidationValidityPeriod: 365, PricingAssetType: cstypes.PricingAssetType_COIN, PricingAsset: "uvna",
	}))
	from := ctx.BlockTime().Add(-time.Hour)
	rootID, err := a.ParticipantKeeper.CreateParticipant(ctx, pptypes.Participant{
		Role: pptypes.ParticipantRole_ECOSYSTEM, SchemaId: 2, CorporationId: 1, Did: "did:example:root2",
		EffectiveFrom: &from, OpState: pptypes.OnboardingState_VALIDATED, ValidationFees: 1000,
	})
	require.NoError(t, err)
	fund := sdk.NewCoins(sdk.NewInt64Coin("uvna", 1_000_000))
	require.NoError(t, a.BankKeeper.MintCoins(ctx, minttypes.ModuleName, fund))
	require.NoError(t, a.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, e.pX, fund))

	_, err = e.des.GrantOperatorAuthorization(ctx, &detypes.MsgGrantOperatorAuthorization{
		Corporation: e.pX.String(), Operator: e.pX.String(), Grantee: e.opX.String(),
		MsgTypes: []string{"/verana.pp.v1.MsgStartParticipantOP", "/verana.pp.v1.MsgRenewParticipantOP"},
	})
	require.NoError(t, err)
	_, err = e.des.GrantOperatorAuthorization(ctx, &detypes.MsgGrantOperatorAuthorization{
		Corporation: e.pE.String(), Operator: e.pE.String(), Grantee: e.opE.String(),
		MsgTypes: []string{"/verana.pp.v1.MsgSetParticipantOPToValidated", "/verana.pp.v1.MsgSlashParticipantTrustDeposit"},
	})
	require.NoError(t, err)

	res, err := e.pps.StartParticipantOP(ctx, &pptypes.MsgStartParticipantOP{
		Corporation: e.pX.String(), Operator: e.opX.String(), Role: pptypes.ParticipantRole_ISSUER,
		ValidatorParticipantId: rootID, Did: "did:example:issuerx2", VsOperator: e.pX.String(),
		VsOperatorAuthzMsgTypes:      []string{pptypes.MsgCreateOrUpdateParticipantSessionTypeURL},
		VsOperatorAuthzWithFeegrant:  true,
		VsOperatorAuthzFeeSpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uvna", 1)),
	})
	require.NoError(t, err)
	id := res.ParticipantId
	validate := &pptypes.MsgSetParticipantOPToValidated{Corporation: e.pE.String(), Operator: e.opE.String(), Id: id}
	_, err = e.pps.SetParticipantOPToValidated(ctx, validate)
	require.NoError(t, err)
	_, err = a.FeeGrantKeeper.GetAllowance(ctx, e.pX, e.pX)
	require.NoError(t, err)

	_, err = e.pps.RenewParticipantOP(ctx, &pptypes.MsgRenewParticipantOP{Corporation: e.pX.String(), Operator: e.opX.String(), Id: id})
	require.NoError(t, err)
	_, err = e.pps.SetParticipantOPToValidated(ctx, validate)
	require.NoError(t, err)
	_, err = a.FeeGrantKeeper.GetAllowance(ctx, e.pX, e.pX)
	require.NoError(t, err)

	p, err := a.ParticipantKeeper.GetParticipantByID(ctx, id)
	require.NoError(t, err)
	require.Positive(t, p.Deposit)
	_, err = e.pps.SlashParticipantTrustDeposit(ctx, &pptypes.MsgSlashParticipantTrustDeposit{Corporation: e.pE.String(), Operator: e.opE.String(), Id: id, Amount: p.Deposit, Reason: "test"})
	require.NoError(t, err)
	e.requireNoSelfGrant(t, ctx)
}

func TestSelfFeeGrant_Genesis(t *testing.T) {
	e := newSelfGrantEnv(t)
	e.selfCreate(t, nil)

	fgGen, err := e.a.FeeGrantKeeper.ExportGenesis(e.ctx)
	require.NoError(t, err)
	require.NoError(t, feegrant.ValidateGenesis(*fgGen))
	deGen, err := e.a.DeKeeper.ExportGenesis(e.ctx)
	require.NoError(t, err)
	require.NoError(t, deGen.Validate())

	a2, ctx2 := feegrantTestApp(t)
	require.NoError(t, a2.FeeGrantKeeper.InitGenesis(ctx2, fgGen))
	_, err = a2.FeeGrantKeeper.GetAllowance(ctx2, e.pX, e.pX)
	require.NoError(t, err)
	require.NoError(t, newDeFeegrantAdapter(a2.FeeGrantKeeper).RevokeAllowance(ctx2, e.pX, e.pX))
	_, err = a2.FeeGrantKeeper.GetAllowance(ctx2, e.pX, e.pX)
	require.ErrorIs(t, err, collections.ErrNotFound)
}
