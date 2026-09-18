package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/core/address"
	storetypes "cosmossdk.io/store/types"
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
	return types.CorporationView{Id: id}, nil
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
	}
}
