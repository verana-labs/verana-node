package gf_test

import (
	"context"
	"time"

	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/gf/types"
)

// Local stub keepers for module-level tests (the module pkg can't import
// x/gf/keeper without breaking the layer boundary; these mirror the runtime
// stubs).

type stubDelegationKeeper struct{}

func (stubDelegationKeeper) CheckDelegateAuthorization(_ context.Context, _ uint64, _ string, _ detypes.MsgType, _ time.Time) error {
	return nil
}
func (stubDelegationKeeper) SetDelegateAcceptedCGFVersion(_ context.Context, _ uint64, _ string, _ uint32) error {
	return nil
}

type stubEcosystemKeeper struct{}

func (*stubEcosystemKeeper) GetEcosystemView(_ context.Context, _ uint64) (types.EcosystemView, bool) {
	return types.EcosystemView{}, false
}
func (*stubEcosystemKeeper) SetEcosystemActiveVersion(_ context.Context, _ uint64, _ uint32, _ time.Time) error {
	return nil
}

type stubCorporationKeeper struct{}

func (*stubCorporationKeeper) ResolveCorporationByPolicyAddress(_ context.Context, _ string) (types.CorporationView, error) {
	return types.CorporationView{}, types.ErrSubjectNotFound
}
func (m *stubCorporationKeeper) ResolveSigner(ctx context.Context, corporation, signer string) (types.CorporationView, bool, error) {
	v, err := m.ResolveCorporationByPolicyAddress(ctx, corporation)
	return v, err == nil && signer == corporation, err
}
func (*stubCorporationKeeper) GetCorporationByPolicyAddress(_ context.Context, _ string) (types.CorporationView, error) {
	return types.CorporationView{}, types.ErrSubjectNotFound
}
func (*stubCorporationKeeper) GetByID(_ context.Context, _ uint64) (types.CorporationView, bool) {
	return types.CorporationView{}, false
}
func (*stubCorporationKeeper) SetActiveVersion(_ context.Context, _ uint64, _ uint32, _ time.Time) error {
	return nil
}
func (*stubCorporationKeeper) SetAcceptedNgfVersion(_ context.Context, _ uint64, _ uint32) error {
	return nil
}
