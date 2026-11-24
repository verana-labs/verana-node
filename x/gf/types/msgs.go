package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/verana-labs/verana-node/util/validation"
)

// validateCorpDelegate checks the corporation and delegate are valid bech32 addresses.
func validateCorpDelegate(corporation, delegate string) error {
	if _, err := sdk.AccAddressFromBech32(corporation); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "corporation: %s", err)
	}
	if _, err := sdk.AccAddressFromBech32(delegate); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "delegate: %s", err)
	}
	return nil
}

// validateSigner checks the delegate path (corporation + delegate) or the
// network path (authority alone).
func validateSigner(corporation, delegate, authority string, ecosystemID uint64) error {
	if authority == "" {
		return validateCorpDelegate(corporation, delegate)
	}
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "authority: %s", err)
	}
	if corporation != "" || delegate != "" || ecosystemID != 0 {
		return errors.Wrap(ErrInvalidSubject, "network path: corporation, delegate and ecosystem_id must be absent")
	}
	return nil
}

// ValidateBasic on MsgUpdateParams: authority must be present.
func (m *MsgUpdateParams) ValidateBasic() error {
	if m.Authority == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "authority is required")
	}
	return m.Params.Validate()
}

// ValidateBasic on MsgAddGovernanceFrameworkDocument.
func (m *MsgAddGovernanceFrameworkDocument) ValidateBasic() error {
	if err := validateSigner(m.Corporation, m.Delegate, m.Authority, m.EcosystemId); err != nil {
		return err
	}
	if m.DocLanguage == "" {
		return errors.Wrap(ErrInvalidLanguage, "doc_language is required")
	}
	if !IsValidBCP47(m.DocLanguage) {
		return errors.Wrap(ErrInvalidLanguage, m.DocLanguage)
	}
	if m.DocUrl == "" {
		return errors.Wrap(ErrInvalidURL, "doc_url is required")
	}
	if !IsValidURL(m.DocUrl) {
		return errors.Wrap(ErrInvalidURL, m.DocUrl)
	}
	if m.DocDigestSri == "" {
		return errors.Wrap(ErrInvalidDigestSRI, "doc_digest_sri is required")
	}
	if !validation.IsValidDigestSRI(m.DocDigestSri) {
		return errors.Wrap(ErrInvalidDigestSRI, m.DocDigestSri)
	}
	if m.Version < 1 {
		return errors.Wrap(ErrInvalidVersion, "version must be >= 1")
	}
	return nil
}

// ValidateBasic on MsgIncreaseActiveGovernanceFrameworkVersion.
func (m *MsgIncreaseActiveGovernanceFrameworkVersion) ValidateBasic() error {
	if err := validateSigner(m.Corporation, m.Delegate, m.Authority, m.EcosystemId); err != nil {
		return err
	}
	if m.AcceptanceGrace <= 0 {
		return errors.Wrap(ErrAcceptanceGraceTooShort, "acceptance_grace is required")
	}
	return nil
}

// ValidateBasic on MsgAcceptGovernanceFrameworkVersion.
func (m *MsgAcceptGovernanceFrameworkVersion) ValidateBasic() error {
	if err := validateCorpDelegate(m.Corporation, m.Delegate); err != nil {
		return err
	}
	if (m.EcosystemId != 0) == m.Network {
		return errors.Wrap(ErrInvalidSubject, "exactly one of ecosystem_id and network must be given")
	}
	if m.Version < 1 {
		return errors.Wrap(ErrInvalidVersion, "version must be >= 1")
	}
	return nil
}

// ValidateBasic on MsgAcceptCorporationGovernanceFramework.
func (m *MsgAcceptCorporationGovernanceFramework) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Delegate); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "delegate: %s", err)
	}
	if m.CorporationId == 0 {
		return errors.Wrap(ErrInvalidSubject, "corporation_id is required")
	}
	if m.Version < 1 {
		return errors.Wrap(ErrInvalidVersion, "version must be >= 1")
	}
	return nil
}
