package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/verana-labs/verana-node/x/xr/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	if err := k.Params.Set(ctx, genState.Params); err != nil {
		return err
	}

	for i := range genState.ExchangeRates {
		xr := &genState.ExchangeRates[i]
		if err := k.ExchangeRates.Set(ctx, xr.Id, xr.Record()); err != nil {
			return fmt.Errorf("failed to set exchange rate %d: %w", xr.Id, err)
		}
	}

	if genState.NextExchangeRateId > 0 {
		if err := k.Counter.Set(ctx, types.CounterKeyExchangeRate, genState.NextExchangeRateId); err != nil {
			return fmt.Errorf("failed to set exchange rate counter: %w", err)
		}
	}

	for i := range genState.ExchangeRateAuthorizations {
		authz := &genState.ExchangeRateAuthorizations[i]
		feeder, err := k.addressCodec.StringToBytes(authz.Feeder)
		if err != nil {
			return err
		}
		xr, err := k.ExchangeRates.Get(ctx, authz.XrId)
		if err != nil {
			return fmt.Errorf("exchange rate authorization references unknown xr_id %d: %w", authz.XrId, err)
		}
		if authz.MaxDeviationBps == 0 && k.isGenesisPair(ctx, &xr) {
			return fmt.Errorf("exchange rate authorization (%d, %s) of the genesis exchange rate needs max_deviation_bps", authz.XrId, authz.Feeder)
		}
		if err := k.ExchangeRateAuthorizations.Set(ctx, collections.Join(authz.XrId, sdk.AccAddress(feeder)), authz.Record()); err != nil {
			return fmt.Errorf("failed to set exchange rate authorization (%d, %s): %w", authz.XrId, authz.Feeder, err)
		}
	}

	return nil
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	var err error

	genesis := types.DefaultGenesis()
	genesis.Params, err = k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	err = k.ExchangeRates.Walk(ctx, nil, func(id uint64, xr types.ExchangeRateRecord) (bool, error) {
		genesis.ExchangeRates = append(genesis.ExchangeRates, xr.ExchangeRate(id))
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to export exchange rates: %w", err)
	}

	nextID, err := k.Counter.Get(ctx, types.CounterKeyExchangeRate)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, fmt.Errorf("failed to get exchange rate counter: %w", err)
	}
	genesis.NextExchangeRateId = nextID

	err = k.ExchangeRateAuthorizations.Walk(ctx, nil, func(key collections.Pair[uint64, sdk.AccAddress], authz types.ExchangeRateAuthorizationRecord) (bool, error) {
		feeder, err := k.addressCodec.BytesToString(key.K2())
		if err != nil {
			return true, err
		}
		genesis.ExchangeRateAuthorizations = append(genesis.ExchangeRateAuthorizations, authz.Authorization(key.K1(), feeder))
		return false, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to export exchange rate authorizations: %w", err)
	}

	return genesis, nil
}
