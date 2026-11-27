package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/vs/types"
)

func TestParams_DefaultsMatchGLO(t *testing.T) {
	p := types.DefaultParams()
	require.Equal(t, 180*24*time.Hour, p.VidRotationMaxTransition)
	require.Equal(t, 24*time.Hour, p.VsAccountFeePeriod)
	require.NoError(t, p.Validate())
}

func TestParams_Validate(t *testing.T) {
	tests := []struct {
		name  string
		edit  func(p *types.Params)
		valid bool
	}{
		{"default", func(p *types.Params) {}, true},
		{"deposits zero", func(p *types.Params) { p.VsRegistrationDeposit, p.EcosystemCreationDeposit = 0, 0 }, true},
		{"deposits set", func(p *types.Params) { p.VsRegistrationDeposit, p.EcosystemCreationDeposit = 500, 10000 }, true},
		{"vid_rotation_max_transition zero", func(p *types.Params) { p.VidRotationMaxTransition = 0 }, false},
		{"vid_rotation_max_transition negative", func(p *types.Params) { p.VidRotationMaxTransition = -time.Second }, false},
		{"vs_account_fee_period zero", func(p *types.Params) { p.VsAccountFeePeriod = 0 }, false},
		{"vs_account_fee_period negative", func(p *types.Params) { p.VsAccountFeePeriod = -time.Hour }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := types.DefaultParams()
			tc.edit(&p)
			if tc.valid {
				require.NoError(t, p.Validate())
			} else {
				require.ErrorIs(t, p.Validate(), types.ErrInvalidParams)
			}
		})
	}
}
