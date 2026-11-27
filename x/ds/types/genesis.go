package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(gs.ValidatorBlocks))
	for _, vb := range gs.ValidatorBlocks {
		addr, err := sdk.ValAddressFromBech32(vb.ValidatorAddress)
		if err != nil {
			return fmt.Errorf("invalid validator address %q: %w", vb.ValidatorAddress, err)
		}
		if _, dup := seen[string(addr)]; dup {
			return fmt.Errorf("duplicate validator blocks for %s", vb.ValidatorAddress)
		}
		seen[string(addr)] = struct{}{}
		if vb.MemberBlocks == 0 || vb.SignedBlocks > vb.MemberBlocks || vb.MemberBlocks > gs.EpochBlocks {
			return fmt.Errorf("validator blocks of %s must satisfy signed_blocks <= member_blocks <= epoch_blocks with member_blocks > 0", vb.ValidatorAddress)
		}
	}
	return nil
}
