package keeper_test

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
	feegrant "cosmossdk.io/x/feegrant"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
	"github.com/verana-labs/verana-node/x/de/keeper"
	module "github.com/verana-labs/verana-node/x/de/module"
	"github.com/verana-labs/verana-node/x/de/types"
)

// mockCorpKeeper backs AUTHZ-CHECK-5 in MOD-DE tests. It resolves any signing
// account by default (permissive); add an address to unregistered to exercise
// the ErrCorporationNotRegistered abort path.
type mockCorpKeeper struct {
	unregistered map[string]bool
}

func newMockCorpKeeper() *mockCorpKeeper {
	return &mockCorpKeeper{unregistered: map[string]bool{}}
}

func (m *mockCorpKeeper) ResolveCorporationByPolicyAddress(_ context.Context, addr string) (types.CorporationView, error) {
	if m.unregistered[addr] {
		return types.CorporationView{}, cotypes.ErrCorporationNotRegistered
	}
	return types.CorporationView{Id: 1, PolicyAddress: addr}, nil
}

func (m *mockCorpKeeper) ResolveCorporationByID(_ context.Context, id uint64) (types.CorporationView, error) {
	return types.CorporationView{Id: id, PolicyAddress: corpPolicyAddr(id)}, nil
}

// corpPolicyAddr is the deterministic policy_address of corporation id in tests.
func corpPolicyAddr(id uint64) string {
	return sdk.AccAddress([]byte(fmt.Sprintf("corp_policy_%08d", id))).String()
}

// memFeegrantKeeper is an in-memory x/feegrant keeper with the SDK's rules:
// a grant cannot overwrite an existing one, revoke and get fail when absent.
// Allowances are stored by pointer, so Accept mutates them in place like
// UseGrantedFees does.
type memFeegrantKeeper struct {
	grants map[string]feegrant.FeeAllowanceI
}

func newMemFeegrantKeeper() *memFeegrantKeeper {
	return &memFeegrantKeeper{grants: map[string]feegrant.FeeAllowanceI{}}
}

func feegrantKey(granter, grantee sdk.AccAddress) string {
	return granter.String() + "/" + grantee.String()
}

func (m *memFeegrantKeeper) GrantAllowance(_ context.Context, granter, grantee sdk.AccAddress, allowance feegrant.FeeAllowanceI) error {
	if _, ok := m.grants[feegrantKey(granter, grantee)]; ok {
		return fmt.Errorf("fee allowance already exists")
	}
	m.grants[feegrantKey(granter, grantee)] = allowance
	return nil
}

func (m *memFeegrantKeeper) RevokeAllowance(_ context.Context, granter, grantee sdk.AccAddress) error {
	if _, ok := m.grants[feegrantKey(granter, grantee)]; !ok {
		return fmt.Errorf("fee-grant not found")
	}
	delete(m.grants, feegrantKey(granter, grantee))
	return nil
}

func (m *memFeegrantKeeper) GetAllowance(_ context.Context, granter, grantee sdk.AccAddress) (feegrant.FeeAllowanceI, error) {
	a, ok := m.grants[feegrantKey(granter, grantee)]
	if !ok {
		return nil, fmt.Errorf("fee-grant not found")
	}
	return a, nil
}

// mockParticipantKeeper backs AUTHZ-CHECK-3 step 1 and the MOD-DE-MSG-5-5 scan
// in MOD-DE tests. Every id is an active participant unless overridden in
// views, or listed in missing.
type mockParticipantKeeper struct {
	views   map[uint64]types.ParticipantView
	missing map[uint64]bool
}

func newMockParticipantKeeper() *mockParticipantKeeper {
	return &mockParticipantKeeper{views: map[uint64]types.ParticipantView{}, missing: map[uint64]bool{}}
}

func (m *mockParticipantKeeper) ViewParticipant(_ context.Context, id uint64) (types.ParticipantView, bool) {
	if m.missing[id] {
		return types.ParticipantView{}, false
	}
	if v, ok := m.views[id]; ok {
		return v, true
	}
	return types.ParticipantView{Active: true}, true
}

type fixture struct {
	ctx          context.Context
	keeper       keeper.Keeper
	addressCodec address.Codec
	corpKeeper   *mockCorpKeeper
	participants *mockParticipantKeeper
	feegrants    *memFeegrantKeeper
}

func initFixture(t *testing.T) *fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority := authtypes.NewModuleAddress(types.GovModuleName)

	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
	)

	// AUTHZ-CHECK-5: wire a permissive corporation keeper (the app wires the real
	// MOD-CO keeper post-construction via the depinject cycle break).
	corpKeeper := newMockCorpKeeper()
	k.SetCorporationKeeper(corpKeeper)
	participants := newMockParticipantKeeper()
	k.SetParticipantKeeper(participants)
	feegrants := newMemFeegrantKeeper()
	k.SetFeegrantKeeper(feegrants)

	// Initialize params
	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return &fixture{
		ctx:          ctx,
		keeper:       k,
		addressCodec: addressCodec,
		corpKeeper:   corpKeeper,
		participants: participants,
		feegrants:    feegrants,
	}
}
