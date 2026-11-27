package types_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/gf/types"
)

func TestParams_DefaultsMatchGLO(t *testing.T) {
	p := types.DefaultParams()
	require.Equal(t, 30*24*time.Hour, p.GfAcceptanceGraceMin)
	require.NoError(t, p.Validate())
}

func TestParams_Validate(t *testing.T) {
	require.ErrorIs(t, types.NewParams(0).Validate(), types.ErrInvalidParams)
	require.ErrorIs(t, types.NewParams(-time.Second).Validate(), types.ErrInvalidParams)
	require.NoError(t, types.NewParams(time.Second).Validate())
}
