package types

import (
	"fmt"
	"time"
)

// DefaultVsOperatorFeePeriod is the default cycle of the aggregate VS operator
// fee allowance.
const DefaultVsOperatorFeePeriod = 24 * time.Hour

// NewParams creates a new Params instance.
func NewParams(vsOperatorFeePeriod time.Duration) Params {
	return Params{VsOperatorFeePeriod: vsOperatorFeePeriod}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams(DefaultVsOperatorFeePeriod)
}

// Validate validates the set of params.
func (p Params) Validate() error {
	if p.VsOperatorFeePeriod <= 0 {
		return fmt.Errorf("vs_operator_fee_period must be positive")
	}
	return nil
}
