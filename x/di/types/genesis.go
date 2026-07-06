package types

import "fmt"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:  DefaultParams(),
		Digests: []Digest{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	seen := make(map[string]struct{}, len(gs.Digests))
	for _, d := range gs.Digests {
		if err := ValidateDigestString(d.Digest); err != nil {
			return err
		}
		if d.Created.IsZero() {
			return fmt.Errorf("created timestamp is required for digest %s", d.Digest)
		}
		if _, dup := seen[d.Digest]; dup {
			return fmt.Errorf("duplicate digest found: %s", d.Digest)
		}
		seen[d.Digest] = struct{}{}
	}
	return nil
}
