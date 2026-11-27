package types

import (
	"context"
	"time"

	detypes "github.com/verana-labs/verana-node/x/de/types"
)

// DelegationKeeper is the minimum surface MOD-GF needs from x/de: AUTHZ-CHECK-1
// and the delegate's acceptance of the corporation's framework (MOD-GF-MSG-4).
// The DE keeper concrete type satisfies it so depinject can auto-wire it.
type DelegationKeeper interface {
	CheckDelegateAuthorization(ctx context.Context, corporationID uint64, delegate string, msgType detypes.MsgType, now time.Time) error
	SetDelegateAcceptedCGFVersion(ctx context.Context, corporationID uint64, delegate string, version uint32) error
}

// EcosystemView is the read shape MOD-GF needs to validate ecosystem subjects.
// `CorporationID` is the uint64 FK to the controlling Corporation.
// `Language` is the ecosystem's primary language.
// `ActiveVersion` is the ecosystem's current active GF version.
type EcosystemView struct {
	Id                 uint64
	CorporationID      uint64
	Language           string
	ActiveVersion      uint32
	AcceptanceDeadline *time.Time
}

// EcosystemKeeper is the minimum surface MOD-GF needs for ecosystem-targeted
// GF ops. Provided by x/ec via EcAsGFEcosystemKeeper after the TR→EC rename
// (issue #305). `CorporationID` on the returned view is the real FK to the
// controlling Corporation; subject-controller checks compare against it.
type EcosystemKeeper interface {
	GetEcosystemView(ctx context.Context, ecosystemID uint64) (EcosystemView, bool)
	SetEcosystemActiveVersion(ctx context.Context, ecosystemID uint64, newVersion uint32, acceptanceDeadline time.Time) error
}

// CorporationView is the read shape MOD-GF needs about a Corporation subject.
// `Id` is the canonical uint64 FK target everywhere in VPR.
// `PolicyAddress` is the on-chain account that signs on the Corporation's behalf.
type CorporationView struct {
	Id                 uint64
	PolicyAddress      string
	Language           string
	ActiveVersion      uint32
	AcceptedNgfVersion uint32
	AcceptanceDeadline *time.Time
}

// CorporationKeeper is the minimum surface MOD-GF needs for:
//   - AUTHZ-CHECK-5 resolution: ResolveCorporationByPolicyAddress takes the
//     signing account (= policy_address), returns the registered Corporation
//     (its id, language, active_version) after GF-CHECK-1, and MUST be called at
//     the entry of every delegable MOD-GF Msg. ResolveSigner does the same for
//     the message's corporation and reports the group-proposal path.
//   - Subject lookups by id: GetByID returns a Corporation by its uint64 id (used
//     by the query layer for the active_only filter).
//   - GetCorporationByPolicyAddress is AUTHZ-CHECK-5 without GF-CHECK-1, for
//     MOD-GF-MSG-3 with `network` true.
//   - Active version mutation: SetActiveVersion sets `co.active_version` and
//     `co.acceptance_deadline` per MOD-GF-MSG-2-3.
//   - SetAcceptedNgfVersion records the corporation's acceptance of the network
//     governance framework (MOD-GF-MSG-3-3).
type CorporationKeeper interface {
	ResolveCorporationByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, error)
	ResolveSigner(ctx context.Context, corporation, signer string) (CorporationView, bool, error)
	GetCorporationByPolicyAddress(ctx context.Context, policyAddress string) (CorporationView, error)
	GetByID(ctx context.Context, corporationID uint64) (CorporationView, bool)
	SetActiveVersion(ctx context.Context, corporationID uint64, newVersion uint32, acceptanceDeadline time.Time) error
	SetAcceptedNgfVersion(ctx context.Context, corporationID uint64, version uint32) error
}
