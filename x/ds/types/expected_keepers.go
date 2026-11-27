package types

import (
	"context"

	"cosmossdk.io/core/address"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
)

type BankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BlockedAddr(addr sdk.AccAddress) bool
}

type StakingKeeper interface {
	GetAllValidators(ctx context.Context) ([]stakingtypes.Validator, error)
	ValidatorAddressCodec() address.Codec
}

type ExchangeRateKeeper interface {
	GetPrice(ctx context.Context, baseAssetType cstypes.PricingAssetType, baseAsset string, quoteAssetType cstypes.PricingAssetType, quoteAsset string, amount math.Int) (math.Int, error)
	// GenesisRate fails while the genesis exchange rate is disabled or expired.
	GenesisRate(ctx context.Context) (uint64, uint32, error)
}

// TrustDepositKeeper provides main_fiat_currency.
type TrustDepositKeeper interface {
	MainFiatCurrency(ctx context.Context) string
}
