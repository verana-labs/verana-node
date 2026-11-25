package types

import (
	"context"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	tdtypes "github.com/verana-labs/verana-node/x/td/types"
)

type BankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BlockedAddr(addr sdk.AccAddress) bool
}

type StakingKeeper interface {
	BondDenom(ctx context.Context) (string, error)
	GetAllValidators(ctx context.Context) ([]stakingtypes.Validator, error)
	ValidatorAddressCodec() address.Codec
}

type ExchangeRateKeeper interface {
	GetPrice(ctx context.Context, baseAssetType cstypes.PricingAssetType, baseAsset string, quoteAssetType cstypes.PricingAssetType, quoteAsset string, amount string) (string, error)
}

// TrustDepositKeeper provides main_fiat_currency.
type TrustDepositKeeper interface {
	GetParams(ctx context.Context) tdtypes.Params
}
