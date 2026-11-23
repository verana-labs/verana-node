package types

import (
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultTwapWindow is a placeholder; the governance framework defines twap_window.
const DefaultTwapWindow = time.Hour

// NewParams creates a new Params instance.
func NewParams(twapWindow time.Duration) Params {
	return Params{TwapWindow: twapWindow}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(DefaultTwapWindow)
}

// Validate checks [MOD-XR-MSG-6-2-1].
func (p Params) Validate() error {
	if p.TwapWindow <= 0 {
		return errorsmod.Wrap(ErrInvalidParams, "twap_window must be positive")
	}
	return nil
}

// EventAttributes lists every parameter value for the update event.
func (p Params) EventAttributes() []sdk.Attribute {
	return []sdk.Attribute{sdk.NewAttribute("twap_window", p.TwapWindow.String())}
}
