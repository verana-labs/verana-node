package types

import (
	"cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/verana-labs/verana-node/util/validation"
)

// validateSigner checks the signer, and the corporation off the network path,
// where corporation and ecosystem_id must be absent.
func validateSigner(corporation, signer string, network bool, ecosystemID uint64) error {
	if _, err := sdk.AccAddressFromBech32(signer); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "signer: %s", err)
	}
	if network {
		if corporation != "" || ecosystemID != 0 {
			return errors.Wrap(ErrInvalidSubject, "network path: corporation and ecosystem_id must be absent")
		}
		return nil
	}
	if _, err := sdk.AccAddressFromBech32(corporation); err != nil {
		return errors.Wrapf(sdkerrors.ErrInvalidAddress, "corporation: %s", err)
	}
	return nil
}

// ValidateBasic on MsgUpdateParams: authority must be present.
func (m *MsgUpdateParams) ValidateBasic() error {
	if m.Signer == "" {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, "signer is required")
	}
	return m.Params.Validate()
}

// ValidateBasic on MsgAddGovernanceFrameworkDocument.
func (m *MsgAddGovernanceFrameworkDocument) ValidateBasic() error {
	if err := validateSigner(m.Corporation, m.Signer, m.Network, m.EcosystemId); err != nil {
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
	return validateSigner(m.Corporation, m.Signer, m.Network, m.EcosystemId)
}
