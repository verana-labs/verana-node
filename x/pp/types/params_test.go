package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/pp/types"
)

func TestParams_DefaultsMatchGLO(t *testing.T) {
	p := types.DefaultParams()
	require.Equal(t, uint32(8), p.AccreditationMaxEntriesPerParticipation)
	require.NoError(t, p.Validate())
}

func TestParams_Validate(t *testing.T) {
	for _, tc := range []struct {
		value uint32
		valid bool
	}{
		{0, false}, {1, false}, {2, true}, {8, true}, {64, true}, {65, false},
	} {
		p := types.NewParams(tc.value)
		if tc.valid {
			require.NoError(t, p.Validate(), tc.value)
		} else {
			require.ErrorIs(t, p.Validate(), types.ErrInvalidParams, tc.value)
		}
	}
}
