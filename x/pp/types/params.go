package types

import (
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultAccreditationMaxEntriesPerParticipation = uint32(8)
	MinAccreditationMaxEntriesPerParticipation     = uint32(2)
	MaxAccreditationMaxEntriesPerParticipation     = uint32(64)
)

// NewParams creates a new Params instance.
func NewParams(accreditationMaxEntriesPerParticipation uint32) Params {
	return Params{AccreditationMaxEntriesPerParticipation: accreditationMaxEntriesPerParticipation}
}

// DefaultParams returns the GLO defaults.
func DefaultParams() Params {
	return NewParams(DefaultAccreditationMaxEntriesPerParticipation)
}

// Validate checks [MOD-AC-MSG-11-2-1].
func (p Params) Validate() error {
	v := p.AccreditationMaxEntriesPerParticipation
	if v < MinAccreditationMaxEntriesPerParticipation || v > MaxAccreditationMaxEntriesPerParticipation {
		return errorsmod.Wrapf(ErrInvalidParams, "accreditation_max_entries_per_participation must be from 2 to 64: %d", v)
	}
	return nil
}

// EventAttributes lists every parameter value for the update event.
func (p Params) EventAttributes() []sdk.Attribute {
	return []sdk.Attribute{
		sdk.NewAttribute("accreditation_max_entries_per_participation", strconv.FormatUint(uint64(p.AccreditationMaxEntriesPerParticipation), 10)),
	}
}
