package types_test

import (
	"testing"

	"cosmossdk.io/math"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/td/types"
)

func TestParams_ValidateRejectsNilDec(t *testing.T) {
	require.Error(t, types.Params{}.Validate())

	p := types.DefaultParams()
	p.TrustDepositRate = math.LegacyDec{}
	require.Error(t, p.Validate())
}
