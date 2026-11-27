package keeper_test

import (
	"context"
	"errors"
	"time"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/ec/types"
	gftypes "github.com/verana-labs/verana-node/x/gf/types"
)

// mockDelegation: configurable AUTHZ-CHECK result. Default returns nil (auth granted).
type mockDelegation struct {
	err       error
	calls     int
	gotCorpID uint64
}

func (m *mockDelegation) CheckDelegateAuthorization(_ context.Context, corporationID uint64, _ string, _ detypes.MsgType, _ time.Time) error {
	m.calls++
	m.gotCorpID = corporationID
	return m.err
}

// mockCorporation: AUTHZ-CHECK-5 resolver. Pre-program signer → CorporationView.
// An unregistered addr fails with cotypes.ErrCorporationNotRegistered.
type mockCorporation struct {
	byAddr    map[string]types.CorporationView
	didOwners map[string]uint64 // Corporation.did -> corp id
}

func newMockCorporation() *mockCorporation {
	return &mockCorporation{byAddr: map[string]types.CorporationView{}, didOwners: map[string]uint64{}}
}

func (m *mockCorporation) register(addr string, id uint64) {
	m.byAddr[addr] = types.CorporationView{Id: id, PolicyAddress: addr, Language: "en", ActiveVersion: 1}
}

func (m *mockCorporation) registerDID(did string, corpID uint64) {
	m.didOwners[did] = corpID
}

func (m *mockCorporation) ResolveCorporationByPolicyAddress(_ context.Context, addr string) (types.CorporationView, error) {
	v, ok := m.byAddr[addr]
	if !ok {
		return types.CorporationView{}, cotypes.ErrCorporationNotRegistered
	}
	return v, nil
}

func (m *mockCorporation) ResolveSigner(ctx context.Context, corporation, signer string) (types.CorporationView, bool, error) {
	v, err := m.ResolveCorporationByPolicyAddress(ctx, corporation)
	return v, err == nil && signer == corporation, err
}

func (m *mockCorporation) ResolveDIDOwner(_ context.Context, did string) (uint64, bool, error) {
	if id, ok := m.didOwners[did]; ok {
		return id, true, nil
	}
	return 0, false, nil
}

func (m *mockCorporation) GetByID(_ context.Context, id uint64) (types.CorporationView, bool) {
	for _, v := range m.byAddr {
		if v.Id == id {
			return v, true
		}
	}
	return types.CorporationView{}, false
}

// mockGF: cross-module gf surface. Captures CreateInitialGFVersionForEcosystem
// and ListVersionsByEcosystem call args + lets tests preprogram list responses
// and errors.
type mockGF struct {
	createErr   error
	createCalls int
	createArgs  struct {
		ecID                           uint64
		language, docURL, docDigestSRI string
	}

	listResp  []gftypes.GovernanceFrameworkVersionWithDocs
	listErr   error
	listCalls int
	listArgs  struct {
		ecID          uint64
		language      string
		activeVersion uint32
		activeOnly    bool
		preferredLang string
	}
}

func (m *mockGF) CreateInitialGFVersionForEcosystem(_ context.Context, ecID uint64, language, docURL, docDigestSRI string) error {
	m.createCalls++
	m.createArgs.ecID = ecID
	m.createArgs.language = language
	m.createArgs.docURL = docURL
	m.createArgs.docDigestSRI = docDigestSRI
	return m.createErr
}

func (m *mockGF) ListVersionsByEcosystem(_ context.Context, ecID uint64, language string, activeVersion uint32, activeOnly bool, preferredLang string) ([]gftypes.GovernanceFrameworkVersionWithDocs, error) {
	m.listCalls++
	m.listArgs.ecID = ecID
	m.listArgs.language = language
	m.listArgs.activeVersion = activeVersion
	m.listArgs.activeOnly = activeOnly
	m.listArgs.preferredLang = preferredLang
	return m.listResp, m.listErr
}

// errAuthDenied is a convenience for tests simulating DE denial.
var errAuthDenied = errors.New("authorization denied")

// Compile-time interface assertions.
var (
	_ types.DelegationKeeper  = (*mockDelegation)(nil)
	_ types.CorporationKeeper = (*mockCorporation)(nil)
	_ types.GFKeeper          = (*mockGF)(nil)
)
