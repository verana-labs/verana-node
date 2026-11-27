package app

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	protov2 "google.golang.org/protobuf/proto"

	gfv1 "github.com/verana-labs/verana-node/api/verana/gf/v1"
	gftypes "github.com/verana-labs/verana-node/x/gf/types"
)

// The two GF messages are signed by `delegate`, or by `authority` (the council
// account) on the network path.
func TestGovernanceFrameworkSigners(t *testing.T) {
	a, _ := feegrantTestApp(t)
	council := sdk.AccAddress([]byte("council_____________"))
	delegate := sdk.AccAddress([]byte("delegate____________"))

	for name, tc := range map[string]struct {
		gogo   sdk.Msg
		pulsar protov2.Message
		want   sdk.AccAddress
	}{
		"add document, delegate": {
			&gftypes.MsgAddGovernanceFrameworkDocument{Signer: delegate.String()},
			&gfv1.MsgAddGovernanceFrameworkDocument{Signer: delegate.String()}, delegate,
		},
		"add document, council": {
			&gftypes.MsgAddGovernanceFrameworkDocument{Signer: council.String()},
			&gfv1.MsgAddGovernanceFrameworkDocument{Signer: council.String()}, council,
		},
		"increase version, delegate": {
			&gftypes.MsgIncreaseActiveGovernanceFrameworkVersion{Signer: delegate.String()},
			&gfv1.MsgIncreaseActiveGovernanceFrameworkVersion{Signer: delegate.String()}, delegate,
		},
		"increase version, council": {
			&gftypes.MsgIncreaseActiveGovernanceFrameworkVersion{Signer: council.String()},
			&gfv1.MsgIncreaseActiveGovernanceFrameworkVersion{Signer: council.String()}, council,
		},
	} {
		t.Run(name, func(t *testing.T) {
			signers, _, err := a.AppCodec().GetMsgV1Signers(tc.gogo)
			require.NoError(t, err)
			require.Equal(t, [][]byte{tc.want}, signers)

			signers, err = a.TxConfig().SigningContext().GetSigners(tc.pulsar)
			require.NoError(t, err)
			require.Equal(t, [][]byte{tc.want}, signers)
		})
	}
}
