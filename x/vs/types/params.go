package types

import (
	"strconv"
	"time"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// placeholders; the governance framework defines both deposits
	DefaultVsRegistrationDeposit    = uint64(0)
	DefaultEcosystemCreationDeposit = uint64(0)
	DefaultVidRotationMaxTransition = 180 * 24 * time.Hour
	DefaultVsAccountFeePeriod       = 24 * time.Hour
)

// NewParams creates a new Params instance.
func NewParams(vsRegistrationDeposit, ecosystemCreationDeposit uint64, vidRotationMaxTransition, vsAccountFeePeriod time.Duration) Params {
	return Params{
		VsRegistrationDeposit:    vsRegistrationDeposit,
		EcosystemCreationDeposit: ecosystemCreationDeposit,
		VidRotationMaxTransition: vidRotationMaxTransition,
		VsAccountFeePeriod:       vsAccountFeePeriod,
	}
}

// DefaultParams returns the GLO defaults.
func DefaultParams() Params {
	return NewParams(
		DefaultVsRegistrationDeposit,
		DefaultEcosystemCreationDeposit,
		DefaultVidRotationMaxTransition,
		DefaultVsAccountFeePeriod,
	)
}

// Validate checks [MOD-VS-MSG-4-2-1]; the deposits are unsigned, so never below 0.
func (p Params) Validate() error {
	if p.VidRotationMaxTransition <= 0 {
		return errorsmod.Wrap(ErrInvalidParams, "vid_rotation_max_transition must be positive")
	}
	if p.VsAccountFeePeriod <= 0 {
		return errorsmod.Wrap(ErrInvalidParams, "vs_account_fee_period must be positive")
	}
	return nil
}

// EventAttributes lists every parameter value for the update event.
func (p Params) EventAttributes() []sdk.Attribute {
	return []sdk.Attribute{
		sdk.NewAttribute("vs_registration_deposit", strconv.FormatUint(p.VsRegistrationDeposit, 10)),
		sdk.NewAttribute("ecosystem_creation_deposit", strconv.FormatUint(p.EcosystemCreationDeposit, 10)),
		sdk.NewAttribute("vid_rotation_max_transition", p.VidRotationMaxTransition.String()),
		sdk.NewAttribute("vs_account_fee_period", p.VsAccountFeePeriod.String()),
	}
}
