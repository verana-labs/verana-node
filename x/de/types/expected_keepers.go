package types

import (
	"context"
	"time"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant"
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

// CorporationView is the read shape MOD-DE needs about a Corporation: its id,
// its policy_address and its elected sponsor (0 when none).
type CorporationView struct {
	Id                   uint64
	PolicyAddress        string
	SponsorCorporationId uint64
}

// CorporationKeeper backs AUTHZ-CHECK-5 for MOD-DE's delegable Grant/Revoke
// messages: resolve the signing `corporation` policy_address to its registered
// Corporation, or abort with ErrCorporationNotRegistered (referencing
// MOD-CO-MSG-1). Wired post-construction via Keeper.SetCorporationKeeper to
// break the MOD-DE ↔ MOD-CO depinject cycle.
type CorporationKeeper interface {
	ResolveCorporationByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, error)
	ResolveCorporationByID(ctx context.Context, id uint64) (CorporationView, error)
	// GetCorporationByPolicyAddress is the AUTHZ-CHECK-5 lookup without GF-CHECK-1.
	GetCorporationByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, bool, error)
	// ClearSponsor clears corporationID's election if it names sponsorID.
	ClearSponsor(ctx context.Context, corporationID, sponsorID uint64) (bool, error)
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

// NetworkFeeKeeper exposes the keys of GlobalVariables.max_gas_per_msg_type.
type NetworkFeeKeeper interface {
	MaxGasMsgTypeURLs(ctx context.Context) ([]string, error)
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

// GovernanceFrameworkKeeper runs GF-CHECK-2 for a delegate (AUTHZ-CHECK-1 step
// 4). Wired post-construction via Keeper.SetGovernanceFrameworkKeeper: MOD-GF
// depends on MOD-DE.
type GovernanceFrameworkKeeper interface {
	CheckDelegateCGFAcceptance(ctx context.Context, corporationID uint64, authz *DelegateAuthorization, now time.Time) error
}

// VerifiableServiceView is what AUTHZ-CHECK-3 and AUTHZ-CHECK-4 read of a
// VerifiableService entry; the calling method resolves it.
type VerifiableServiceView struct {
	Id            uint64
	CorporationId uint64
	VsAccount     string
	MsgTypes      []MsgType
	Period        *time.Duration
	FeeSpendLimit sdk.Coins
}

// AccreditationView is what AUTHZ-CHECK-3 reads of its primary entry.
type AccreditationView struct {
	Id     uint64
	Active bool
}

// VsAccountBudget is the running operation budget of the VS account of a
// VerifiableService entry.
type VsAccountBudget struct {
	SpendLimit     sdk.Coins
	RemainingSpend sdk.Coins
	Expiration     *time.Time
}

// VsAccountBudgetStore holds the VsAccountBudget rows keyed by vs_id that
// AUTHZ-CHECK-3 steps 5 and 6 read and write. Get returns
// collections.ErrNotFound when the entry has no budget.
type VsAccountBudgetStore interface {
	Get(ctx context.Context, vsID uint64) (VsAccountBudget, error)
	Set(ctx context.Context, vsID uint64, budget VsAccountBudget) error
}
