package types

import (
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultCredentialSchemaSchemaMaxSize                                    = uint64(8192)
	DefaultCredentialSchemaIssuerGrantorAccreditationValidityPeriodMaxDays   = uint32(3650)
	DefaultCredentialSchemaVerifierGrantorAccreditationValidityPeriodMaxDays = uint32(3650)
	DefaultCredentialSchemaIssuerAccreditationValidityPeriodMaxDays          = uint32(3650)
	DefaultCredentialSchemaVerifierAccreditationValidityPeriodMaxDays        = uint32(3650)
	DefaultCredentialSchemaHolderAccreditationValidityPeriodMaxDays          = uint32(3650)
	DefaultCredentialSchemaPresentationValidityPeriodMaxDays                 = uint32(3650)
)

// NewParams creates a new Params instance.
func NewParams(
	schemaMaxSize uint64,
	issuerGrantorMaxDays uint32,
	verifierGrantorMaxDays uint32,
	issuerMaxDays uint32,
	verifierMaxDays uint32,
	holderMaxDays uint32,
	presentationMaxDays uint32,
) Params {
	return Params{
		CredentialSchemaSchemaMaxSize:                                    schemaMaxSize,
		CredentialSchemaIssuerGrantorAccreditationValidityPeriodMaxDays:   issuerGrantorMaxDays,
		CredentialSchemaVerifierGrantorAccreditationValidityPeriodMaxDays: verifierGrantorMaxDays,
		CredentialSchemaIssuerAccreditationValidityPeriodMaxDays:          issuerMaxDays,
		CredentialSchemaVerifierAccreditationValidityPeriodMaxDays:        verifierMaxDays,
		CredentialSchemaHolderAccreditationValidityPeriodMaxDays:          holderMaxDays,
		CredentialSchemaPresentationValidityPeriodMaxDays:                 presentationMaxDays,
	}
}

// DefaultParams returns the GLO defaults.
func DefaultParams() Params {
	return NewParams(
		DefaultCredentialSchemaSchemaMaxSize,
		DefaultCredentialSchemaIssuerGrantorAccreditationValidityPeriodMaxDays,
		DefaultCredentialSchemaVerifierGrantorAccreditationValidityPeriodMaxDays,
		DefaultCredentialSchemaIssuerAccreditationValidityPeriodMaxDays,
		DefaultCredentialSchemaVerifierAccreditationValidityPeriodMaxDays,
		DefaultCredentialSchemaHolderAccreditationValidityPeriodMaxDays,
		DefaultCredentialSchemaPresentationValidityPeriodMaxDays,
	)
}

// Validate checks [MOD-CS-MSG-4-2-1]: every value MUST be > 0.
func (p Params) Validate() error {
	if p.CredentialSchemaSchemaMaxSize == 0 {
		return errorsmod.Wrap(ErrInvalidParams, "credential_schema_schema_max_size must be positive")
	}
	for name, v := range map[string]uint32{
		"credential_schema_issuer_grantor_accreditation_validity_period_max_days":   p.CredentialSchemaIssuerGrantorAccreditationValidityPeriodMaxDays,
		"credential_schema_verifier_grantor_accreditation_validity_period_max_days": p.CredentialSchemaVerifierGrantorAccreditationValidityPeriodMaxDays,
		"credential_schema_issuer_accreditation_validity_period_max_days":           p.CredentialSchemaIssuerAccreditationValidityPeriodMaxDays,
		"credential_schema_verifier_accreditation_validity_period_max_days":         p.CredentialSchemaVerifierAccreditationValidityPeriodMaxDays,
		"credential_schema_holder_accreditation_validity_period_max_days":           p.CredentialSchemaHolderAccreditationValidityPeriodMaxDays,
		"credential_schema_presentation_validity_period_max_days":                   p.CredentialSchemaPresentationValidityPeriodMaxDays,
	} {
		if v == 0 {
			return errorsmod.Wrapf(ErrInvalidParams, "%s must be positive", name)
		}
	}
	return nil
}

// EventAttributes lists every parameter value for the update event.
func (p Params) EventAttributes() []sdk.Attribute {
	u32 := func(v uint32) string { return strconv.FormatUint(uint64(v), 10) }
	return []sdk.Attribute{
		sdk.NewAttribute("credential_schema_schema_max_size", strconv.FormatUint(p.CredentialSchemaSchemaMaxSize, 10)),
		sdk.NewAttribute("credential_schema_issuer_grantor_accreditation_validity_period_max_days", u32(p.CredentialSchemaIssuerGrantorAccreditationValidityPeriodMaxDays)),
		sdk.NewAttribute("credential_schema_verifier_grantor_accreditation_validity_period_max_days", u32(p.CredentialSchemaVerifierGrantorAccreditationValidityPeriodMaxDays)),
		sdk.NewAttribute("credential_schema_issuer_accreditation_validity_period_max_days", u32(p.CredentialSchemaIssuerAccreditationValidityPeriodMaxDays)),
		sdk.NewAttribute("credential_schema_verifier_accreditation_validity_period_max_days", u32(p.CredentialSchemaVerifierAccreditationValidityPeriodMaxDays)),
		sdk.NewAttribute("credential_schema_holder_accreditation_validity_period_max_days", u32(p.CredentialSchemaHolderAccreditationValidityPeriodMaxDays)),
		sdk.NewAttribute("credential_schema_presentation_validity_period_max_days", u32(p.CredentialSchemaPresentationValidityPeriodMaxDays)),
	}
}
