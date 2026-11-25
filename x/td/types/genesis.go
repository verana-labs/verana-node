package types

import (
	"fmt"
	"math"
)

// DefaultIndex is the default global index
const DefaultIndex uint64 = 1

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:        DefaultParams(),
		TrustDeposits: []TrustDepositRecord{},
		Dust:          "",
		TuPegValue:    DefaultTuPegValueGenesis,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	if gs.PendingTuDecayRate != nil {
		if err := ValidateTuDecayRate(*gs.PendingTuDecayRate); err != nil {
			return err
		}
	}
	if err := gs.validateTuIndex(); err != nil {
		return err
	}

	// Check for duplicate corporation_ids in trust deposits
	corporationSet := make(map[uint64]struct{}, len(gs.TrustDeposits))

	for i, td := range gs.TrustDeposits {
		if td.CorporationId == 0 {
			return fmt.Errorf("invalid corporation_id (zero) at index %d", i)
		}

		if _, exists := corporationSet[td.CorporationId]; exists {
			return fmt.Errorf("duplicate trust deposit for corporation_id: %d", td.CorporationId)
		}
		corporationSet[td.CorporationId] = struct{}{}

		if td.Deposit < td.Refunded {
			return fmt.Errorf("refunded amount exceeds deposit for corporation_id %d: %d > %d",
				td.CorporationId, td.Refunded, td.Deposit)
		}

		if td.RepaidDeposit > td.SlashedDeposit {
			return fmt.Errorf("repaid_deposit exceeds slashed_deposit for corporation_id %d: %d > %d",
				td.CorporationId, td.RepaidDeposit, td.SlashedDeposit)
		}

		if td.Share.IsNil() || td.Share.IsNegative() {
			return fmt.Errorf("invalid share for corporation_id %d: %s", td.CorporationId, td.Share.String())
		}
	}

	return nil
}

// validateTuIndex checks [TUG]: the index only declines from tu_peg_value_genesis, and only once an epoch is closed.
func (gs GenesisState) validateTuIndex() error {
	peg, genesisPeg := gs.TuPegValue, gs.Params.TuPegValueGenesis
	switch {
	case peg.IsNil() || peg.IsNegative() || peg.GT(genesisPeg):
		return fmt.Errorf("tu_peg_value must be in [0, %s]: %s", genesisPeg, decString(peg))
	case gs.LastClosedEpoch == nil && !peg.Equal(genesisPeg):
		return fmt.Errorf("tu_peg_value must equal tu_peg_value_genesis %s while no epoch is closed: %s", genesisPeg, peg)
	case gs.LastClosedEpoch != nil && gs.EpochOrigin == nil:
		return fmt.Errorf("last_closed_epoch requires epoch_origin")
	case gs.EpochOrigin != nil && gs.EpochOrigin.Unix() < 0:
		return fmt.Errorf("epoch_origin must not be before 1970: %s", gs.EpochOrigin)
	case gs.LastClosedEpoch != nil && *gs.LastClosedEpoch >= uint64(math.MaxInt64/int64(gs.Params.EpochLength)):
		return fmt.Errorf("last_closed_epoch out of range for epoch_length %s: %d", gs.Params.EpochLength, *gs.LastClosedEpoch)
	}
	return nil
}
