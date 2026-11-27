package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/core/address"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	cotypes "github.com/verana-labs/verana-node/x/co/types"
	"github.com/verana-labs/verana-node/x/di/keeper"
	module "github.com/verana-labs/verana-node/x/di/module"
	"github.com/verana-labs/verana-node/x/di/types"
)

// mockCorpKeeper backs AUTHZ-CHECK-5 in MOD-DI tests. It resolves any signing
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

func (m *mockCorpKeeper) ResolveSigner(ctx context.Context, corporation, signer string) (types.CorporationView, bool, error) {
	v, err := m.ResolveCorporationByPolicyAddress(ctx, corporation)
	return v, err == nil && signer == corporation, err
}

type fixture struct {
	ctx          context.Context
	keeper       keeper.Keeper
	addressCodec address.Codec
	corpKeeper   *mockCorpKeeper
}

func initFixture(t *testing.T) *fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	storeService := runtime.NewKVStoreService(storeKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	authority := authtypes.NewModuleAddress("council")

	corpKeeper := newMockCorpKeeper()
	k := keeper.NewKeeper(
		storeService,
		encCfg.Codec,
		addressCodec,
		authority,
		&keepertest.MockDelegationKeeper{},
		corpKeeper,
	)

	return &fixture{
		ctx:          ctx,
		keeper:       k,
		addressCodec: addressCodec,
		corpKeeper:   corpKeeper,
	}
}
