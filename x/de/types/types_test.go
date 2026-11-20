package types_test

import (
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

func TestIsOwnPolicyAddress(t *testing.T) {
	policy := sdk.AccAddress([]byte("corp_policy_________")).String()
	other := sdk.AccAddress([]byte("operator____________")).String()

	require.True(t, types.IsOwnPolicyAddress(policy, policy))
	require.True(t, types.IsOwnPolicyAddress(policy, strings.ToUpper(policy)))
	require.False(t, types.IsOwnPolicyAddress(policy, other))
	require.False(t, types.IsOwnPolicyAddress(policy, ""))
	require.False(t, types.IsOwnPolicyAddress(policy, "not-an-address"))
	require.False(t, types.IsOwnPolicyAddress("", ""))
}
