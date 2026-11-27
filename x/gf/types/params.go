package types

import (
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const DefaultGfAcceptanceGraceMin = 30 * 24 * time.Hour

func NewParams(gfAcceptanceGraceMin time.Duration) Params {
	return Params{GfAcceptanceGraceMin: gfAcceptanceGraceMin}
}

// DefaultParams returns the GLO defaults.
func DefaultParams() Params {
	return NewParams(DefaultGfAcceptanceGraceMin)
}

// Validate checks [MOD-GF-MSG-5-2-1].
func (p Params) Validate() error {
	if p.GfAcceptanceGraceMin <= 0 {
		return errorsmod.Wrap(ErrInvalidParams, "gf_acceptance_grace_min must be positive")
	}
	return nil
}

// EventAttributes lists every parameter value for the update event.
func (p Params) EventAttributes() []sdk.Attribute {
	return []sdk.Attribute{sdk.NewAttribute("gf_acceptance_grace_min", p.GfAcceptanceGraceMin.String())}
}
