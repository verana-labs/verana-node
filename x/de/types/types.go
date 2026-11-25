package types

import (
	"fmt"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// IsOwnPolicyAddress reports whether account is the corporation's
// policy_address. Decoded addresses are compared: bech32 accepts both cases.
func IsOwnPolicyAddress(policyAddress, account string) bool {
	policy, err := sdk.AccAddressFromBech32(policyAddress)
	if err != nil {
		return false
	}
	addr, err := sdk.AccAddressFromBech32(account)
	return err == nil && addr.Equals(policy)
}

// ValidateBasic performs stateless validation on MsgGrantDelegateAuthorization.
func (msg *MsgGrantDelegateAuthorization) ValidateBasic() error {
	// corporation is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// delegate is the signer; mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Delegate); err != nil {
		return fmt.Errorf("invalid delegate address: %w", err)
	}

	// grantee is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Grantee); err != nil {
		return fmt.Errorf("invalid grantee address: %w", err)
	}

	if err := ValidateMsgTypes(msg.MsgTypes, DelegateAuthorizationMsgTypes); err != nil {
		return err
	}

	// authz_spend_limit if specified must be valid and all-positive
	if len(msg.AuthzSpendLimit) > 0 && !msg.AuthzSpendLimit.IsValid() {
		return fmt.Errorf("invalid authz_spend_limit")
	}
	if len(msg.AuthzSpendLimit) > 0 && !msg.AuthzSpendLimit.IsAllPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "authz_spend_limit amounts must be positive")
	}

	// authz_spend_limit_period if specified must be a valid (positive) period;
	// ignored if authz_spend_limit is not set [MOD-DE-MSG-3-2]
	if len(msg.AuthzSpendLimit) > 0 && msg.AuthzSpendLimitPeriod != nil && *msg.AuthzSpendLimitPeriod <= 0 {
		return fmt.Errorf("authz_spend_limit_period must be a positive duration")
	}

	// [MOD-DE-MSG-3-2] a spend period requires an expiration window
	// (DelegateAuthorization invariant: period set => expiration set).
	if len(msg.AuthzSpendLimit) > 0 && msg.AuthzSpendLimitPeriod != nil && msg.Expiration == nil {
		return fmt.Errorf("expiration must be set when authz_spend_limit_period is set")
	}

	// feegrant_spend_limit is ignored when with_feegrant is false (spec MOD-DE-MSG-3-2).

	// feegrant_spend_limit if specified must be valid and all-positive (only relevant if with_feegrant)
	if msg.WithFeegrant && len(msg.FeegrantSpendLimit) > 0 && !msg.FeegrantSpendLimit.IsValid() {
		return fmt.Errorf("invalid feegrant_spend_limit")
	}
	if len(msg.FeegrantSpendLimit) > 0 && !msg.FeegrantSpendLimit.IsAllPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "feegrant_spend_limit amounts must be positive")
	}

	// feegrant_spend_limit_period if specified must be a valid (positive) period;
	// ignored if feegrant_spend_limit is not set or with_feegrant is false [MOD-DE-MSG-3-2]
	if msg.WithFeegrant && len(msg.FeegrantSpendLimit) > 0 && msg.FeegrantSpendLimitPeriod != nil && *msg.FeegrantSpendLimitPeriod <= 0 {
		return fmt.Errorf("feegrant_spend_limit_period must be a positive duration")
	}

	return nil
}

// ValidateBasic performs stateless validation on MsgRevokeDelegateAuthorization.
func (msg *MsgRevokeDelegateAuthorization) ValidateBasic() error {
	// corporation is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Corporation); err != nil {
		return fmt.Errorf("invalid corporation address: %w", err)
	}

	// delegate is the signer; mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Delegate); err != nil {
		return fmt.Errorf("invalid delegate address: %w", err)
	}

	// grantee is mandatory
	if _, err := sdk.AccAddressFromBech32(msg.Grantee); err != nil {
		return fmt.Errorf("invalid grantee address: %w", err)
	}

	return nil
}
