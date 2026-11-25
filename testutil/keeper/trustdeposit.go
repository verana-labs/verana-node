package keeper

import (
	"context"
	"testing"

	"cosmossdk.io/log/v2"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/store/v2"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
	"github.com/verana-labs/verana-node/x/td/keeper"
	"github.com/verana-labs/verana-node/x/td/types"
)

// MockTDCorporationKeeper backs AUTHZ-CHECK-5 in MOD-TD tests. It resolves any
// signing account by default (permissive) so pre-existing positive-path tests
// pass; add an address to Unregistered to exercise the
// ErrCorporationNotRegistered abort path.
type MockTDCorporationKeeper struct {
	Unregistered map[string]bool
	ids          map[string]uint64
	nextID       uint64
}

func NewMockTDCorporationKeeper() *MockTDCorporationKeeper {
	return &MockTDCorporationKeeper{Unregistered: map[string]bool{}, ids: map[string]uint64{}, nextID: 1}
}

// IDFor returns the stable corporation_id assigned to addr (assigning a fresh
// id on first use). Trust deposits are keyed by corporation_id, so tests use
// this to look up records by the same id the keeper resolves internally.
func (m *MockTDCorporationKeeper) IDFor(addr string) uint64 {
	if m.ids == nil {
		m.ids = map[string]uint64{}
	}
	if id, ok := m.ids[addr]; ok {
		return id
	}
	if m.nextID == 0 {
		m.nextID = 1
	}
	id := m.nextID
	m.ids[addr] = id
	m.nextID++
	return id
}

func (m *MockTDCorporationKeeper) ResolveCorporationByPolicyAddress(_ context.Context, addr string) (types.CorporationView, error) {
	if m.Unregistered[addr] {
		return types.CorporationView{}, cotypes.ErrCorporationNotRegistered
	}
	return types.CorporationView{Id: m.IDFor(addr), PolicyAddress: addr}, nil
}

// MockGenesisRateKeeper serves the genesis exchange rate entry; Err makes it unavailable.
type MockGenesisRateKeeper struct {
	Rate  uint64
	Scale uint32
	Err   error
}

func (m *MockGenesisRateKeeper) GenesisRate(context.Context) (uint64, uint32, error) {
	return m.Rate, m.Scale, m.Err
}

// MockDistributionKeeper records the epochs paid out; OnPayout runs inside each call.
type MockDistributionKeeper struct {
	Epochs   []uint64
	OnPayout func(ctx context.Context, epoch uint64) error
}

func (m *MockDistributionKeeper) PayoutEpoch(ctx context.Context, epoch uint64) error {
	m.Epochs = append(m.Epochs, epoch)
	if m.OnPayout != nil {
		return m.OnPayout(ctx, epoch)
	}
	return nil
}

func newTrustdepositKeeper(
	t testing.TB,
	delegationKeeper types.DelegationKeeper,
	coKeeper types.CorporationKeeper,
	xrKeeper types.ExchangeRateKeeper,
	dsKeeper types.DistributionKeeper,
) (keeper.Keeper, sdk.Context) {
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	require.NoError(t, stateStore.LoadLatestVersion())

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	authority := authtypes.NewModuleAddress(govtypes.ModuleName)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		log.NewNopLogger(),
		authority.String(),
		NewMockBankKeeper(),
		delegationKeeper,
		coKeeper,
		xrKeeper,
		dsKeeper,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{}, false, log.NewNopLogger())
	require.NoError(t, k.SetParams(ctx, types.DefaultParams()))
	require.NoError(t, k.TuPegValue.Set(ctx, types.DefaultTuPegValueGenesis))
	require.NoError(t, k.EpochOrigin.Set(ctx, ctx.BlockTime().Unix()))

	return k, ctx
}

func TrustdepositKeeper(t testing.TB) (keeper.Keeper, sdk.Context) {
	return newTrustdepositKeeper(t, &MockDelegationKeeper{}, NewMockTDCorporationKeeper(), &MockGenesisRateKeeper{}, nil)
}

// TrustdepositKeeperWithDelegation creates a keeper with a MockDelegationKeeper for AUTHZ testing.
// The returned MockDelegationKeeper can be configured to return errors via ErrToReturn.
func TrustdepositKeeperWithDelegation(t testing.TB) (keeper.Keeper, sdk.Context, *MockDelegationKeeper) {
	delegationKeeper := &MockDelegationKeeper{}
	k, ctx := newTrustdepositKeeper(t, delegationKeeper, NewMockTDCorporationKeeper(), &MockGenesisRateKeeper{}, nil)
	return k, ctx, delegationKeeper
}

// TrustdepositKeeperWithCorp creates a keeper exposing the AUTHZ-CHECK-5
// CorporationKeeper mock so a test can mark a signing account unregistered and
// assert the ErrCorporationNotRegistered abort path.
func TrustdepositKeeperWithCorp(t testing.TB) (keeper.Keeper, sdk.Context, *MockDelegationKeeper, *MockTDCorporationKeeper) {
	delegationKeeper := &MockDelegationKeeper{}
	coKeeper := NewMockTDCorporationKeeper()
	k, ctx := newTrustdepositKeeper(t, delegationKeeper, coKeeper, &MockGenesisRateKeeper{}, nil)
	return k, ctx, delegationKeeper, coKeeper
}

// TrustdepositKeeperWithIndex exposes the exchange rate and distribution mocks of the trust unit index.
func TrustdepositKeeperWithIndex(t testing.TB) (keeper.Keeper, sdk.Context, *MockGenesisRateKeeper, *MockDistributionKeeper) {
	xrKeeper := &MockGenesisRateKeeper{}
	dsKeeper := &MockDistributionKeeper{}
	k, ctx := newTrustdepositKeeper(t, &MockDelegationKeeper{}, NewMockTDCorporationKeeper(), xrKeeper, dsKeeper)
	return k, ctx, xrKeeper, dsKeeper
}
