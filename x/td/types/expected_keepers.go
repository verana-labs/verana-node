package types

import (
	"context"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	detypes "github.com/verana-labs/verana-node/x/de/types"
)

// AccountKeeper defines the expected interface for the Account module.
type AccountKeeper interface {
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule, recipientModule string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	BurnCoins(ctx context.Context, name string, amt sdk.Coins) error
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

// MintKeeper defines the expected interface for the Mint module.
// Mint keeper has Params collections.Item[types.Params] where Params has BlocksPerYear field
type MintKeeper interface {
	// GetBlocksPerYear returns the blocks_per_year parameter from mint module
	GetBlocksPerYear(ctx context.Context) (uint64, error)
}

// DelegationKeeper defines the expected interface for the Delegation (DE) module.
// Used to perform [AUTHZ-CHECK] for (authority, delegate) pairs.
type DelegationKeeper interface {
	CheckDelegateAuthorization(ctx context.Context, corporationID uint64, delegate string, msgType detypes.MsgType, now time.Time) error
	ConsumeDelegateSpend(ctx context.Context, corporationID uint64, delegate string, msgType detypes.MsgType, now time.Time, amount sdk.Coins) error
}

// CorporationView is the read shape MOD-TD needs about a Corporation subject
// for AUTHZ-CHECK-5: the signing corporation policy_address resolved to its id.
type CorporationView struct {
	Id            uint64
	PolicyAddress string
}

// CorporationKeeper backs AUTHZ-CHECK-5 (with GF-CHECK-1) for MOD-TD delegable
// messages, and the plain account lookup of the trust deposit adjustments other
// modules call for any corporation. Both abort with ErrCorporationNotRegistered
// (referencing MOD-CO-MSG-1) for an unregistered account.
type CorporationKeeper interface {
	ResolveCorporationByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, error)
	GetCorporationByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, error)
}
