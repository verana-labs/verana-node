package types

import (
	"context"
	"time"

	"cosmossdk.io/core/address"
	feegrant "cosmossdk.io/x/feegrant"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AuthKeeper defines the expected interface for the Auth module.
type AuthKeeper interface {
	AddressCodec() address.Codec
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// Methods imported from account should be defined here
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	// Methods imported from bank should be defined here
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}

// CorporationView is the read shape MOD-DE needs about a Corporation subject
// for AUTHZ-CHECK-5: the signing `corporation` policy_address resolved to its id.
type CorporationView struct {
	Id            uint64
	PolicyAddress string
}

// CorporationKeeper backs AUTHZ-CHECK-5 for MOD-DE's delegable Grant/Revoke
// messages: resolve the signing `corporation` policy_address to its registered
// Corporation, or abort with ErrCorporationNotRegistered (referencing
// MOD-CO-MSG-1). Wired post-construction via Keeper.SetCorporationKeeper to
// break the MOD-DE ↔ MOD-CO depinject cycle.
type CorporationKeeper interface {
	ResolveCorporationByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, error)
	ResolveCorporationByID(ctx context.Context, id uint64) (CorporationView, error)
}

// FeegrantKeeper is the cosmos x/feegrant keeper subset MOD-DE uses to realize
// FeeGrant entities as on-chain allowances (spec Delegation Module note: corp-
// paid fees are charged by x/feegrant during fee processing). Wired post-
// construction via Keeper.SetFeegrantKeeper.
type FeegrantKeeper interface {
	GrantAllowance(ctx context.Context, granter, grantee sdk.AccAddress, allowance feegrant.FeeAllowanceI) error
	RevokeAllowance(ctx context.Context, granter, grantee sdk.AccAddress) error
	GetAllowance(ctx context.Context, granter, grantee sdk.AccAddress) (feegrant.FeeAllowanceI, error)
}

// ParticipantView is what MOD-DE reads about a Participant entry: its glossary
// state at block time and its window end.
type ParticipantView struct {
	Active         bool
	Future         bool
	EffectiveUntil *time.Time
}

// ParticipantKeeper backs AUTHZ-CHECK-3 step 1 and the MOD-DE-MSG-5-5 scan.
// Wired post-construction via Keeper.SetParticipantKeeper: MOD-PP depends on
// MOD-DE, so MOD-DE cannot import MOD-PP.
type ParticipantKeeper interface {
	ViewParticipant(ctx context.Context, id uint64) (ParticipantView, bool)
}
