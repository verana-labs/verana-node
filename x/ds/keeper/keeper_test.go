package keeper_test

import (
	"context"
	"errors"
	"testing"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/ds/keeper"
	module "github.com/verana-labs/verana-node/x/ds/module"
	"github.com/verana-labs/verana-node/x/ds/types"
	tdtypes "github.com/verana-labs/verana-node/x/td/types"
)

const (
	denom = "uvna"
	fiat  = "USD"
)

var poolAddr = authtypes.NewModuleAddress(types.ModuleName)

type mockBank struct {
	balances map[string]math.Int
	burned   math.Int
	blocked  map[string]bool
}

func (b *mockBank) GetBalance(_ context.Context, addr sdk.AccAddress, d string) sdk.Coin {
	if v, ok := b.balances[addr.String()]; ok {
		return sdk.NewCoin(d, v)
	}
	return sdk.NewCoin(d, math.ZeroInt())
}

func (b *mockBank) SendCoinsFromModuleToAccount(_ context.Context, module string, to sdk.AccAddress, amt sdk.Coins) error {
	from := authtypes.NewModuleAddress(module).String()
	a := amt.AmountOf(denom)
	if b.GetBalance(nil, authtypes.NewModuleAddress(module), denom).Amount.LT(a) {
		return errors.New("insufficient funds")
	}
	b.balances[from] = b.balances[from].Sub(a)
	b.balances[to.String()] = b.GetBalance(nil, to, denom).Amount.Add(a)
	return nil
}

func (b *mockBank) BurnCoins(_ context.Context, module string, amt sdk.Coins) error {
	from := authtypes.NewModuleAddress(module).String()
	b.balances[from] = b.balances[from].Sub(amt.AmountOf(denom))
	b.burned = b.burned.Add(amt.AmountOf(denom))
	return nil
}

func (b *mockBank) BlockedAddr(addr sdk.AccAddress) bool { return b.blocked[addr.String()] }

type mockStaking struct{ validators []stakingtypes.Validator }

func (s *mockStaking) BondDenom(context.Context) (string, error) { return denom, nil }
func (s *mockStaking) GetAllValidators(context.Context) ([]stakingtypes.Validator, error) {
	return s.validators, nil
}
func (s *mockStaking) ValidatorAddressCodec() address.Codec {
	return addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
}

// mockXr converts at rate/10^scale; a nil rate fails every non-zero amount.
type mockXr struct {
	rate  *math.Int
	scale int64
}

func (x *mockXr) GetPrice(_ context.Context, bt cstypes.PricingAssetType, b string, qt cstypes.PricingAssetType, q string, amount string) (string, error) {
	if bt != cstypes.PricingAssetType_FIAT || b != fiat || qt != cstypes.PricingAssetType_COIN || q != denom {
		return "", errors.New("unexpected pair")
	}
	a, ok := math.NewIntFromString(amount)
	if !ok {
		return "", errors.New("invalid amount")
	}
	if a.IsZero() {
		return "0", nil
	}
	if x.rate == nil {
		return "", errors.New("exchange rate is expired")
	}
	return a.Mul(*x.rate).Quo(math.NewIntWithDecimal(1, int(x.scale))).String(), nil
}

type mockTd struct{}

func (mockTd) GetParams(context.Context) tdtypes.Params {
	return tdtypes.Params{MainFiatCurrency: fiat}
}

type fixture struct {
	ctx          sdk.Context
	keeper       keeper.Keeper
	addressCodec address.Codec
	bank         *mockBank
	staking      *mockStaking
	xr           *mockXr
	cdc          codec.Codec
}

func initFixture(t *testing.T) *fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	rate := math.NewInt(2)
	f := &fixture{
		ctx:          ctx,
		addressCodec: addressCodec,
		bank:         &mockBank{balances: map[string]math.Int{}, burned: math.ZeroInt(), blocked: map[string]bool{}},
		staking:      &mockStaking{},
		xr:           &mockXr{rate: &rate},
		cdc:          encCfg.Codec,
	}
	f.keeper = keeper.NewKeeper(runtime.NewKVStoreService(storeKey), encCfg.Codec, addressCodec, authtypes.NewModuleAddress("council"), f.bank, f.staking, f.xr)
	f.keeper.SetTrustDepositKeeper(mockTd{})
	if err := f.keeper.Params.Set(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}
	return f
}

func (f *fixture) authority(t *testing.T) string {
	t.Helper()
	s, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// addValidator seats a validator with the given tokens and returns its
// operator address and consensus address.
func (f *fixture) addValidator(t *testing.T, seed string, tokens int64) (sdk.ValAddress, sdk.ConsAddress) {
	t.Helper()
	pk := ed25519.GenPrivKeyFromSecret([]byte(seed)).PubKey()
	op := sdk.ValAddress(pk.Address())
	anyPk, err := codectypes.NewAnyWithValue(pk)
	if err != nil {
		t.Fatal(err)
	}
	f.staking.validators = append(f.staking.validators, stakingtypes.Validator{
		OperatorAddress: op.String(),
		ConsensusPubkey: anyPk,
		Tokens:          math.NewInt(tokens),
	})
	return op, sdk.ConsAddress(pk.Address())
}

func (f *fixture) fundPool(amount int64) {
	f.bank.balances[poolAddr.String()] = math.NewInt(amount)
}

func (f *fixture) balance(addr sdk.AccAddress) math.Int {
	return f.bank.GetBalance(f.ctx, addr, denom).Amount
}
