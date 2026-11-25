package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/ds/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	// authority is the council account.
	authority []byte

	bank    types.BankKeeper
	staking types.StakingKeeper
	xr      types.ExchangeRateKeeper
	// td is wired after construction: x/td depends on this keeper.
	td *trustDepositKeeperRef

	Schema          collections.Schema
	Params          collections.Item[types.Params]
	ValidatorBlocks collections.Map[sdk.ValAddress, types.ValidatorBlocks]
	EpochBlocks     collections.Item[uint64]
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,
	bank types.BankKeeper,
	staking types.StakingKeeper,
	xr types.ExchangeRateKeeper,
) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,
		bank:         bank,
		staking:      staking,
		xr:           xr,
		td:           &trustDepositKeeperRef{},

		Params:          collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		ValidatorBlocks: collections.NewMap(sb, types.ValidatorBlocksKey, "validator_blocks", sdk.ValAddressKey, codec.CollValue[types.ValidatorBlocks](cdc)),
		EpochBlocks:     collections.NewItem(sb, types.EpochBlocksKey, "epoch_blocks", collections.Uint64Value),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

type trustDepositKeeperRef struct{ K types.TrustDepositKeeper }

// SetTrustDepositKeeper wires x/td once both keepers exist.
func (k Keeper) SetTrustDepositKeeper(td types.TrustDepositKeeper) {
	k.td.K = td
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}
