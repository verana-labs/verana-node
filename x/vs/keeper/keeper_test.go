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

	"github.com/verana-labs/verana-node/x/vs/keeper"
	module "github.com/verana-labs/verana-node/x/vs/module"
	"github.com/verana-labs/verana-node/x/vs/types"
)

type fixture struct {
	ctx          context.Context
	keeper       keeper.Keeper
	addressCodec address.Codec
}

func initFixture(t *testing.T) *fixture {
	t.Helper()

	encCfg := moduletestutil.MakeTestEncodingConfig(module.AppModule{})
	addressCodec := addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	ctx := testutil.DefaultContextWithDB(t, storeKey, storetypes.NewTransientStoreKey("transient_test")).Ctx

	k := keeper.NewKeeper(runtime.NewKVStoreService(storeKey), encCfg.Codec, addressCodec, authtypes.NewModuleAddress("council"))
	if err := k.Params.Set(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return &fixture{ctx: ctx, keeper: k, addressCodec: addressCodec}
}

func (f *fixture) authority(t *testing.T) string {
	t.Helper()
	s, err := f.addressCodec.BytesToString(f.keeper.GetAuthority())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
