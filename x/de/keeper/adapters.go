package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/collections"

	"github.com/verana-labs/verana-node/x/de/types"
)

// corpKeeperRef is a stable container for the CorporationKeeper interface,
// indirected behind a pointer so all by-value copies of Keeper (held by the msg
// server, query server, etc.) see the same instance. SetCorporationKeeper
// writes to this container after construction — required because MOD-CO depends
// on MOD-DE's DelegationKeeper while MOD-DE needs MOD-CO for AUTHZ-CHECK-5
// (cycle break per #308).
type corpKeeperRef struct {
	K types.CorporationKeeper
}

// StubCorporationKeeper is the pre-wiring default. If a delegable MOD-DE handler
// somehow runs before SetCorporationKeeper has been invoked, AUTHZ-CHECK-5
// aborts cleanly instead of nil-panicking. Replaced via Keeper.SetCorporationKeeper.
type StubCorporationKeeper struct{}

func (StubCorporationKeeper) ResolveCorporationByPolicyAddress(_ context.Context, policyAddress string) (types.CorporationView, error) {
	return types.CorporationView{}, fmt.Errorf("corporation keeper not wired: cannot resolve signing account %s", policyAddress)
}

func (StubCorporationKeeper) ResolveCorporationByID(_ context.Context, id uint64) (types.CorporationView, error) {
	return types.CorporationView{}, fmt.Errorf("corporation keeper not wired: cannot resolve corporation id %d", id)
}

func (StubCorporationKeeper) GetCorporationByPolicyAddress(_ context.Context, policyAddress string) (types.CorporationView, bool, error) {
	return types.CorporationView{}, false, fmt.Errorf("corporation keeper not wired: cannot look up %s", policyAddress)
}

func (StubCorporationKeeper) ClearSponsor(_ context.Context, corporationID, _ uint64) (bool, error) {
	return false, fmt.Errorf("corporation keeper not wired: cannot clear the sponsor of corporation %d", corporationID)
}

// feegrantKeeperRef holds the FeegrantKeeper behind a shared pointer; nil until wired.
type feegrantKeeperRef struct {
	K types.FeegrantKeeper
}

// participantKeeperRef holds the ParticipantKeeper behind a shared pointer, like
// corpKeeperRef. Wired via SetParticipantKeeper.
type participantKeeperRef struct {
	K types.ParticipantKeeper
}

// StubParticipantKeeper is the pre-wiring default: no entry is ever found, so
// AUTHZ-CHECK-3 step 1 aborts and the recompute skips every record.
type StubParticipantKeeper struct{}

func (StubParticipantKeeper) ViewParticipant(_ context.Context, _ uint64) (types.ParticipantView, bool) {
	return types.ParticipantView{}, false
}

// gfKeeperRef holds the GovernanceFrameworkKeeper behind a shared pointer, like
// corpKeeperRef. Wired via SetGovernanceFrameworkKeeper.
type gfKeeperRef struct {
	K types.GovernanceFrameworkKeeper
}

// StubGovernanceFrameworkKeeper is the pre-wiring default: GF-CHECK-2 aborts
// instead of passing silently.
type StubGovernanceFrameworkKeeper struct{}

func (StubGovernanceFrameworkKeeper) CheckDelegateCGFAcceptance(_ context.Context, corporationID uint64, _ *types.DelegateAuthorization, _ time.Time) error {
	return fmt.Errorf("governance framework keeper not wired: cannot run GF-CHECK-2 for corporation %d", corporationID)
}

// vsAccountBudgetStoreRef holds the VsAccountBudgetStore behind a shared pointer.
// Wired via SetVsAccountBudgetStore.
type vsAccountBudgetStoreRef struct {
	S types.VsAccountBudgetStore
}

// StubVsAccountBudgetStore is the pre-wiring default: no entry has a budget.
type StubVsAccountBudgetStore struct{}

func (StubVsAccountBudgetStore) Get(_ context.Context, _ uint64) (types.VsAccountBudget, error) {
	return types.VsAccountBudget{}, collections.ErrNotFound
}

func (StubVsAccountBudgetStore) Set(_ context.Context, vsID uint64, _ types.VsAccountBudget) error {
	return fmt.Errorf("VS account budget store not wired: cannot write the budget of entry %d", vsID)
}

// networkFeeKeeperRef holds the NetworkFeeKeeper behind a shared pointer.
type networkFeeKeeperRef struct {
	K types.NetworkFeeKeeper
}

// StubNetworkFeeKeeper is the pre-wiring default: covers_fees grants abort.
type StubNetworkFeeKeeper struct{}

func (StubNetworkFeeKeeper) MaxGasMsgTypeURLs(_ context.Context) ([]string, error) {
	return nil, fmt.Errorf("network fee keeper not wired: max_gas_per_msg_type is unavailable")
}
