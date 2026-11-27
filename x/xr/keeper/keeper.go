package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	tdtypes "github.com/verana-labs/verana-node/x/td/types"
	"github.com/verana-labs/verana-node/x/xr/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec
	bankKeeper   types.BankKeeper
	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the council account.
	authority []byte
	// td serves main_fiat_currency; wired post-construction via SetTrustDepositKeeper.
	td *trustDepositRef

	Schema        collections.Schema
	Params        collections.Item[types.Params]
	ExchangeRates collections.Map[uint64, types.ExchangeRateRecord]
	Counter       collections.Map[string, uint64]
	// ExchangeRateAuthorizations is keyed by (xr_id, feeder).
	ExchangeRateAuthorizations collections.Map[collections.Pair[uint64, sdk.AccAddress], types.ExchangeRateAuthorizationRecord]
}

type trustDepositRef struct {
	k types.TrustDepositKeeper
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,
	bankKeeper types.BankKeeper,
) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		bankKeeper:   bankKeeper,
		authority:    authority,
		td:           &trustDepositRef{},

		Params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		ExchangeRates: collections.NewMap(
			sb,
			types.ExchangeRateKey,
			"exchange_rate",
			collections.Uint64Key,
			codec.CollValue[types.ExchangeRateRecord](cdc),
		),
		Counter: collections.NewMap(
			sb,
			types.CounterKey,
			"counter",
			collections.StringKey,
			collections.Uint64Value,
		),
		ExchangeRateAuthorizations: collections.NewMap(
			sb,
			types.ExchangeRateAuthorizationKey,
			"exchange_rate_authorization",
			collections.PairKeyCodec(collections.Uint64Key, sdk.AccAddressKey),
			codec.CollValue[types.ExchangeRateAuthorizationRecord](cdc),
		),
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// SetTrustDepositKeeper wires the keeper owning main_fiat_currency; every keeper copy shares it.
func (k Keeper) SetTrustDepositKeeper(td types.TrustDepositKeeper) {
	k.td.k = td
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}

func (k Keeper) checkAuthority(signer string) error {
	bz, err := k.addressCodec.StringToBytes(signer)
	if err != nil || !bytes.Equal(k.authority, bz) {
		expected, _ := k.addressCodec.BytesToString(k.authority)
		return errorsmod.Wrapf(types.ErrInvalidSigner, "invalid signer; expected %s, got %s", expected, signer)
	}
	return nil
}

// GetNextID returns the next auto-incremented ID for the given entity type.
func (k Keeper) GetNextID(ctx context.Context, entityType string) (uint64, error) {
	currentID, err := k.Counter.Get(ctx, entityType)
	if err != nil {
		currentID = 0
	}

	nextID := currentID + 1
	if err := k.Counter.Set(ctx, entityType, nextID); err != nil {
		return 0, fmt.Errorf("failed to set counter: %w", err)
	}

	return nextID, nil
}

// findExchangeRate scans the entries, a few by design, for the pair.
func (k Keeper) findExchangeRate(ctx context.Context, baseType cstypes.PricingAssetType, base string, quoteType cstypes.PricingAssetType, quote string) (uint64, types.ExchangeRateRecord, error) {
	var (
		id  uint64
		rec types.ExchangeRateRecord
	)
	err := k.ExchangeRates.Walk(ctx, nil, func(key uint64, r types.ExchangeRateRecord) (bool, error) {
		if r.BaseAssetType == baseType && r.BaseAsset == base && r.QuoteAssetType == quoteType && r.QuoteAsset == quote {
			id, rec = key, r
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return 0, rec, err
	}
	if id == 0 {
		return 0, rec, errorsmod.Wrapf(types.ErrExchangeRateNotFound, "%s %s to %s %s", baseType, base, quoteType, quote)
	}
	return id, rec, nil
}

// isGenesisPair reports whether r prices the main fiat currency in the native denom.
func (k Keeper) isGenesisPair(ctx context.Context, r *types.ExchangeRateRecord) bool {
	return r.BaseAssetType == cstypes.PricingAssetType_FIAT && r.BaseAsset == k.td.k.MainFiatCurrency(ctx) &&
		r.QuoteAssetType == cstypes.PricingAssetType_COIN && r.QuoteAsset == tdtypes.BondDenom
}

// checkUsable fails on a disabled or expired entry.
func checkUsable(ctx context.Context, r *types.ExchangeRateRecord) error {
	if !r.State {
		return types.ErrExchangeRateNotActive
	}
	if r.Expires <= sdk.UnwrapSDKContext(ctx).BlockTime().Unix() {
		return types.ErrExchangeRateExpired
	}
	return nil
}

// GenesisRate returns rate and rate_scale of the genesis exchange rate; it fails while disabled or expired.
func (k Keeper) GenesisRate(ctx context.Context) (uint64, uint32, error) {
	r, err := k.genesisEntry(ctx)
	if err != nil {
		return 0, 0, err
	}
	if err := checkUsable(ctx, &r); err != nil {
		return 0, 0, err
	}
	return r.Rate, r.RateScale, nil
}

// LastGenesisRate returns rate and rate_scale of the genesis exchange rate, even if disabled or expired.
func (k Keeper) LastGenesisRate(ctx context.Context) (uint64, uint32, error) {
	r, err := k.genesisEntry(ctx)
	return r.Rate, r.RateScale, err
}

func (k Keeper) genesisEntry(ctx context.Context) (types.ExchangeRateRecord, error) {
	_, r, err := k.findExchangeRate(ctx, cstypes.PricingAssetType_FIAT, k.td.k.MainFiatCurrency(ctx), cstypes.PricingAssetType_COIN, tdtypes.BondDenom)
	return r, err
}

func isNotFound(err error) bool {
	return errors.Is(err, collections.ErrNotFound) || errors.Is(err, types.ErrExchangeRateNotFound)
}
