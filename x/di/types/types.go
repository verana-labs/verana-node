package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// ValidateDigestString checks a digest is non-empty and within the 256-byte
// cap. Shared by the Msg and module-call paths.
//
// [MOD-DI-MSG-1-1] defines digest as an opaque string: the chain stores it as a
// key and never recomputes it against any content, so it imposes no format. A
// party interpreting a credential digest resolves the algorithm from the
// digest_algorithm of the governing CredentialSchema, never from the value.
func ValidateDigestString(digest string) error {
	if digest == "" {
		return ErrDigestEmpty
	}
	if len(digest) > 256 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "digest exceeds maximum length of 256 bytes")
	}
	return nil
}

// ValidateBasic performs stateless validation on MsgStoreDigest.
func (msg *MsgStoreDigest) ValidateBasic() error {
	// authority (corporation) is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return fmt.Errorf("invalid authority address: %w", err)
	}

	// operator is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Operator); err != nil {
		return fmt.Errorf("invalid operator address: %w", err)
	}

	return ValidateDigestString(msg.Digest)
}
