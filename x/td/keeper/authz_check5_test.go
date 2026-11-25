package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	cotypes "github.com/verana-labs/verana-node/x/co/types"
	"github.com/verana-labs/verana-node/x/td/keeper"
	"github.com/verana-labs/verana-node/x/td/types"
)

// TestAuthzCheck5_TrustDeposit verifies AUTHZ-CHECK-5 on the MOD-TD delegable
// messages: an unregistered signing corporation aborts with
// ErrCorporationNotRegistered; a registered one passes the check.
func TestAuthzCheck5_TrustDeposit(t *testing.T) {
	k, ctx, _, coKeeper := keepertest.TrustdepositKeeperWithCorp(t)
	ms := keeper.NewMsgServerImpl(k)
	corp := sdk.AccAddress([]byte("unregistered_corp___")).String()
	delegate := sdk.AccAddress([]byte("delegate____________")).String()

	t.Run("RepaySlashedTrustDeposit: registered corporation passes AUTHZ-CHECK-5", func(t *testing.T) {
		delete(coKeeper.Unregistered, corp)
		_, err := ms.RepaySlashedTrustDeposit(ctx, &types.MsgRepaySlashedTrustDeposit{
			Corporation: corp,
			Delegate:    delegate,
		})
		// Fails later (no trust deposit exists), but NOT with the registration error.
		require.Error(t, err)
		require.NotErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
	})

	t.Run("RepaySlashedTrustDeposit: unregistered corporation aborts", func(t *testing.T) {
		coKeeper.Unregistered[corp] = true
		_, err := ms.RepaySlashedTrustDeposit(ctx, &types.MsgRepaySlashedTrustDeposit{
			Corporation: corp,
			Delegate:    delegate,
		})
		require.ErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
	})
}

// AUTHZ-CHECK-5 runs before AUTHZ-CHECK-1, which receives the resolved id.
func TestAuthzCheck5_TrustDepositOrder(t *testing.T) {
	corp := sdk.AccAddress([]byte("td_corporation______")).String()
	delegate := sdk.AccAddress([]byte("delegate____________")).String()
	k, ctx, del, coKeeper := keepertest.TrustdepositKeeperWithCorp(t)
	ms := keeper.NewMsgServerImpl(k)
	repay := func() error {
		_, err := ms.RepaySlashedTrustDeposit(ctx, &types.MsgRepaySlashedTrustDeposit{Corporation: corp, Delegate: delegate, Deposit: 1})
		return err
	}

	coKeeper.Unregistered[corp] = true
	require.ErrorIs(t, repay(), cotypes.ErrCorporationNotRegistered)
	require.Empty(t, del.DelegateAuthCorpIDs)

	delete(coKeeper.Unregistered, corp)
	_ = repay()
	require.Equal(t, []uint64{coKeeper.IDFor(corp)}, del.DelegateAuthCorpIDs)
}

// Trust deposit adjustments resolve accounts that are not the message's
// corporation, so they never go through AUTHZ-CHECK-5 and GF-CHECK-1.
func TestAuthzCheck5_TrustDepositAdjustLookup(t *testing.T) {
	k, ctx, _, coKeeper := keepertest.TrustdepositKeeperWithCorp(t)
	account := sdk.AccAddress([]byte("other_corporation___")).String()

	require.NoError(t, k.AdjustTrustDeposit(ctx, account, 10, "test"))
	require.Empty(t, coKeeper.Resolved)
}
