package keeper

import (
	"context"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	detypes "github.com/verana-labs/verana-node/x/de/types"
)

// MockExchangeRateKeeper is an identity price oracle for tests: it returns the
// amount unchanged, which matches the native (COIN, uvna) Case A path where no
// conversion happens. Tests exercising TU/COIN/FIAT conversion set their own.
type MockExchangeRateKeeper struct{}

func (MockExchangeRateKeeper) GetPrice(_ context.Context, _ cstypes.PricingAssetType, _ string, _ cstypes.PricingAssetType, _ string, amount string) (string, error) {
	return amount, nil
}

// MockTrustDepositKeeper is a no-op mock satisfying x/cs / x/pp /
// trustDeposit-consumer interfaces in test wiring. Extracted from the
// pre-rename testutil/keeper/trustregistry.go.
type MockTrustDepositKeeper struct{}

func (m *MockTrustDepositKeeper) AdjustTrustDeposit(_ sdk.Context, _ string, _ int64, _ string) error {
	return nil
}

func (m *MockTrustDepositKeeper) AdjustTrustDepositOnBehalf(_ sdk.Context, _ string, _ sdk.AccAddress, _ int64) error {
	return nil
}

func (m *MockTrustDepositKeeper) GetTrustDepositRate(_ sdk.Context) math.LegacyDec {
	v, _ := math.LegacyNewDecFromStr("0")
	return v
}

func (m *MockTrustDepositKeeper) GetUserAgentRewardRate(_ sdk.Context) math.LegacyDec {
	v, _ := math.LegacyNewDecFromStr("0")
	return v
}

func (m *MockTrustDepositKeeper) GetWalletUserAgentRewardRate(_ sdk.Context) math.LegacyDec {
	v, _ := math.LegacyNewDecFromStr("0")
	return v
}

func (m *MockTrustDepositKeeper) BurnEcosystemSlashedTrustDeposit(_ sdk.Context, _ string, _ uint64) error {
	return nil
}

// MockDelegationKeeper is a mock implementation of the DelegationKeeper
// interface used by cs / perm / td / ec tests. By default it allows all
// operator authorizations (ErrToReturn is nil). Set ErrToReturn to simulate
// authorization failures. Extracted from the pre-rename
// testutil/keeper/trustregistry.go.
type MockDelegationKeeper struct {
	ErrToReturn error

	// Per-method overrides take precedence over ErrToReturn when non-nil, so a
	// test can fail one authorization path while another succeeds (e.g. force the
	// operator path to fail to exercise the vs-operator delegation path).
	OperatorAuthErr   error
	VSOperatorAuthErr error

	GrantVSOACalls  []GrantVSOACall
	RevokeVSOACalls []uint64 // participant ids
	UpdateVSOACalls []UpdateVSOACall
}

type GrantVSOACall struct {
	CorporationID uint64
	VsOperator    string
	Record        detypes.ParticipantAuthorizationRecord
}

type UpdateVSOACall struct {
	ParticipantID uint64
	NewExpiration *time.Time
}

func (m *MockDelegationKeeper) Reset() {
	m.GrantVSOACalls = nil
	m.RevokeVSOACalls = nil
	m.UpdateVSOACalls = nil
}

func (m *MockDelegationKeeper) CheckOperatorAuthorization(_ context.Context, _, _, _ string, _ time.Time) error {
	if m.OperatorAuthErr != nil {
		return m.OperatorAuthErr
	}
	return m.ErrToReturn
}

func (m *MockDelegationKeeper) CheckVSOperatorAuthorizationOnParticipant(_ context.Context, _ uint64, _ string, _ uint64, _ string) error {
	if m.VSOperatorAuthErr != nil {
		return m.VSOperatorAuthErr
	}
	return m.ErrToReturn
}

func (m *MockDelegationKeeper) CheckVSOperatorFeeGrant(_ context.Context, _ uint64) error {
	return m.ErrToReturn
}

func (m *MockDelegationKeeper) ConsumeOperatorSpend(_ context.Context, _, _, _ string, _ time.Time, _ sdk.Coins) error {
	return m.ErrToReturn
}

func (m *MockDelegationKeeper) ConsumeRecordSpend(_ context.Context, _ uint64, _ string, _ uint64, _ sdk.Coins) error {
	return m.ErrToReturn
}

func (m *MockDelegationKeeper) ConsumeRecordFeeSpend(_ context.Context, _ uint64, _ string, _ uint64, _ sdk.Coins) error {
	return m.ErrToReturn
}

func (m *MockDelegationKeeper) GrantVSOperatorAuthorization(_ context.Context, corporationID uint64, vsOperator string, record detypes.ParticipantAuthorizationRecord) error {
	m.GrantVSOACalls = append(m.GrantVSOACalls, GrantVSOACall{CorporationID: corporationID, VsOperator: vsOperator, Record: record})
	return m.ErrToReturn
}

func (m *MockDelegationKeeper) RevokeVSOperatorAuthorization(_ context.Context, participantID uint64) error {
	m.RevokeVSOACalls = append(m.RevokeVSOACalls, participantID)
	return m.ErrToReturn
}

func (m *MockDelegationKeeper) UpdateVSOperatorAuthorizationExpiration(_ context.Context, participantID uint64, newExpiration *time.Time) error {
	m.UpdateVSOACalls = append(m.UpdateVSOACalls, UpdateVSOACall{ParticipantID: participantID, NewExpiration: newExpiration})
	return m.ErrToReturn
}
