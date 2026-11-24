package types

import (
	"time"

	errorsmod "cosmossdk.io/errors"
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
