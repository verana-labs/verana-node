package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
	"github.com/verana-labs/verana-node/x/de/keeper"
	"github.com/verana-labs/verana-node/x/de/types"
)

// TestAuthzCheck5_DelegateAuthorization verifies AUTHZ-CHECK-5 on the MOD-DE
// delegable Grant/Revoke messages: an unregistered signing corporation aborts
// with ErrCorporationNotRegistered (the registered path is covered by the
// existing Grant/Revoke happy-path tests via the permissive corp keeper).
func TestAuthzCheck5_DelegateAuthorization(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)

	corp := sdk.AccAddress([]byte("unregistered_corp___")).String()
	grantee := sdk.AccAddress([]byte("grantee_address_____")).String()
	f.corpKeeper.unregistered[corp] = true

	t.Run("GrantDelegateAuthorization: unregistered corporation aborts", func(t *testing.T) {
		_, err := ms.GrantDelegateAuthorization(f.ctx, &types.MsgGrantDelegateAuthorization{
			Corporation: corp,
			Delegate:    corp, // group-proposal path: delegate == corporation
			Grantee:     grantee,
			MsgTypes:    []types.MsgType{mtEcosystem},
		})
		require.ErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
	})

	t.Run("RevokeDelegateAuthorization: unregistered corporation aborts", func(t *testing.T) {
		_, err := ms.RevokeDelegateAuthorization(f.ctx, &types.MsgRevokeDelegateAuthorization{
			Corporation: corp,
			Delegate:    corp, // group-proposal path: delegate == corporation
			Grantee:     grantee,
		})
		require.ErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
	})
}
