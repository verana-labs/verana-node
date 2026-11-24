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

	if len(msg.AuthzSpendLimit) > 0 && !msg.AuthzSpendLimit.IsValid() {
		return errorsmod.Wrap(ErrInvalidSpendLimit, "authz_spend_limit")
	}
	if len(msg.FeegrantSpendLimit) > 0 && !msg.FeegrantSpendLimit.IsValid() {
		return errorsmod.Wrap(ErrInvalidSpendLimit, "feegrant_spend_limit")
	}
	if msg.AuthzSpendLimitPeriod != nil && *msg.AuthzSpendLimitPeriod <= 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "authz_spend_limit_period must be a positive duration")
	}
	if msg.FeegrantSpendLimitPeriod != nil && *msg.FeegrantSpendLimitPeriod <= 0 {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "feegrant_spend_limit_period must be a positive duration")
	}
	if (msg.AuthzSpendLimitPeriod != nil || msg.FeegrantSpendLimitPeriod != nil) && msg.Expiration == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "expiration must be set with a spend limit period")
	}
	// A fee allowance never cycles under an authorization that expires.
	if msg.FeegrantSpendLimitPeriod != nil && (len(msg.AuthzSpendLimit) == 0 || msg.AuthzSpendLimitPeriod == nil) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "feegrant_spend_limit_period requires authz_spend_limit and authz_spend_limit_period")
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
