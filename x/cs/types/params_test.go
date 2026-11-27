package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/cs/types"
)

func TestParams_DefaultsMatchGLO(t *testing.T) {
	p := types.DefaultParams()
	require.Equal(t, uint64(8192), p.CredentialSchemaSchemaMaxSize)
	for _, v := range []uint32{
		p.CredentialSchemaIssuerGrantorAccreditationValidityPeriodMaxDays,
		p.CredentialSchemaVerifierGrantorAccreditationValidityPeriodMaxDays,
		p.CredentialSchemaIssuerAccreditationValidityPeriodMaxDays,
		p.CredentialSchemaVerifierAccreditationValidityPeriodMaxDays,
		p.CredentialSchemaHolderAccreditationValidityPeriodMaxDays,
		p.CredentialSchemaPresentationValidityPeriodMaxDays,
	} {
		require.Equal(t, uint32(3650), v)
	}
	require.NoError(t, p.Validate())
}

func TestParams_Validate(t *testing.T) {
	tests := []struct {
		name string
		edit func(p *types.Params)
	}{
		{"schema_max_size", func(p *types.Params) { p.CredentialSchemaSchemaMaxSize = 0 }},
		{"issuer_grantor", func(p *types.Params) { p.CredentialSchemaIssuerGrantorAccreditationValidityPeriodMaxDays = 0 }},
		{"verifier_grantor", func(p *types.Params) { p.CredentialSchemaVerifierGrantorAccreditationValidityPeriodMaxDays = 0 }},
		{"issuer", func(p *types.Params) { p.CredentialSchemaIssuerAccreditationValidityPeriodMaxDays = 0 }},
		{"verifier", func(p *types.Params) { p.CredentialSchemaVerifierAccreditationValidityPeriodMaxDays = 0 }},
		{"holder", func(p *types.Params) { p.CredentialSchemaHolderAccreditationValidityPeriodMaxDays = 0 }},
		{"presentation", func(p *types.Params) { p.CredentialSchemaPresentationValidityPeriodMaxDays = 0 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := types.DefaultParams()
			tc.edit(&p)
			require.ErrorIs(t, p.Validate(), types.ErrInvalidParams)
		})
	}
}

func TestParams_ValidateNamesFirstInvalidField(t *testing.T) {
	p := types.DefaultParams()
	p.CredentialSchemaIssuerGrantorAccreditationValidityPeriodMaxDays = 0
	p.CredentialSchemaVerifierGrantorAccreditationValidityPeriodMaxDays = 0
	p.CredentialSchemaPresentationValidityPeriodMaxDays = 0
	for i := 0; i < 50; i++ {
		require.ErrorContains(t, p.Validate(), "credential_schema_issuer_grantor_accreditation_validity_period_max_days must be positive")
	}
}
