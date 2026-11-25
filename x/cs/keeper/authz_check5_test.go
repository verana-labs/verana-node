package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	cotypes "github.com/verana-labs/verana-node/x/co/types"
	"github.com/verana-labs/verana-node/x/cs/keeper"
	"github.com/verana-labs/verana-node/x/cs/types"
)

// AUTHZ-CHECK-5 runs first on every MOD-CS message and AUTHZ-CHECK-1 receives
// the corporation id it resolved.
func TestAuthzCheck5_CredentialSchema(t *testing.T) {
	corporation := sdk.AccAddress([]byte("cs_corporation______")).String()
	delegate := sdk.AccAddress([]byte("cs_operator_________")).String()
	msgs := map[string]func(ms types.MsgServer, ctx sdk.Context) error{
		"create": func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.CreateCredentialSchema(ctx, &types.MsgCreateCredentialSchema{Corporation: corporation, Delegate: delegate, EcosystemId: 1})
			return err
		},
		"update": func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.UpdateCredentialSchema(ctx, &types.MsgUpdateCredentialSchema{Corporation: corporation, Delegate: delegate, Id: 1})
			return err
		},
		"archive": func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.ArchiveCredentialSchema(ctx, &types.MsgArchiveCredentialSchema{Corporation: corporation, Delegate: delegate, Id: 1})
			return err
		},
	}
	for name, run := range msgs {
		t.Run(name+": unregistered corporation aborts first", func(t *testing.T) {
			k, _, co, del, ctx := keepertest.CredentialschemaKeeperWithMocks(t)
			co.Unregistered[corporation] = true
			require.ErrorIs(t, run(keeper.NewMsgServerImpl(k), ctx), cotypes.ErrCorporationNotRegistered)
			require.Empty(t, del.DelegateAuthCorpIDs)
		})
		t.Run(name+": AUTHZ-CHECK-1 gets the resolved id", func(t *testing.T) {
			k, ec, _, del, ctx := keepertest.CredentialschemaKeeperWithMocks(t)
			corpID := ec.CreateMockEcosystem(corporation, "did:example:cs") // corporation id == ecosystem id
			_ = run(keeper.NewMsgServerImpl(k), ctx)
			require.Equal(t, []uint64{corpID}, del.DelegateAuthCorpIDs)
		})
	}
}
