package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	cotypes "github.com/verana-labs/verana-node/x/co/types"
	"github.com/verana-labs/verana-node/x/di/keeper"
	"github.com/verana-labs/verana-node/x/di/types"
)

// TestAuthzCheck5_StoreDigest verifies AUTHZ-CHECK-5 on MOD-DI MsgStoreDigest:
// an unregistered signing authority aborts with ErrCorporationNotRegistered
// (the registered/permissive path is covered by TestStoreDigest_HappyPath).
func TestAuthzCheck5_StoreDigest(t *testing.T) {
	mock := &keepertest.MockDelegationKeeper{}
	f := initFixtureWithMock(t, mock)
	ms := keeper.NewMsgServerImpl(f.keeper)

	authority := sdk.AccAddress([]byte("unregistered_corp___")).String()
	delegate := sdk.AccAddress([]byte("delegate_address____")).String()

	f.corpKeeper.unregistered[authority] = true
	_, err := ms.StoreDigest(f.ctx, &types.MsgStoreDigest{
		Authority: authority,
		Delegate:  delegate,
		Digest:    "sha256-abc123def456",
	})
	require.ErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
}

// AUTHZ-CHECK-1 runs after AUTHZ-CHECK-5 and receives the resolved id.
func TestAuthzCheck5_StoreDigestOrder(t *testing.T) {
	mock := &keepertest.MockDelegationKeeper{}
	f := initFixtureWithMock(t, mock)
	ms := keeper.NewMsgServerImpl(f.keeper)
	authority := sdk.AccAddress([]byte("registered_corp_____")).String()
	msg := &types.MsgStoreDigest{
		Authority: authority,
		Delegate:  sdk.AccAddress([]byte("operator_address____")).String(),
		Digest:    "sha256-abc123def456",
	}

	f.corpKeeper.unregistered[authority] = true
	_, err := ms.StoreDigest(f.ctx, msg)
	require.ErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
	require.Empty(t, mock.DelegateAuthCorpIDs)

	delete(f.corpKeeper.unregistered, authority)
	_, err = ms.StoreDigest(f.ctx, msg)
	require.NoError(t, err)
	require.Equal(t, []uint64{1}, mock.DelegateAuthCorpIDs)
}
