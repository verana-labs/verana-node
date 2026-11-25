package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

const testVsID uint64 = 9

func vsAccountEntry(vsAccount string) *types.VerifiableServiceView {
	return &types.VerifiableServiceView{
		Id:            testVsID,
		CorporationId: testCorpID,
		VsAccount:     vsAccount,
		MsgTypes:      []types.MsgType{mtIssuance, types.MsgType_MSG_TYPE_TRIGGER_RESOLVER},
	}
}

func TestCheckVSAccount_PrimaryNotActive(t *testing.T) {
	f := initFixture(t)
	vsAccount := acc("vs_account__________")
	perm := &types.AccreditationView{Id: 4, Active: false}
	err := f.keeper.CheckVSAccountAuthorization(f.ctx, testCorpID, vsAccount, perm, vsAccountEntry(vsAccount), mtIssuance)
	require.ErrorIs(t, err, types.ErrAccreditationNotActive)
}

func TestCheckVSAccount_ActiveOrNullPrimary(t *testing.T) {
	f := initFixture(t)
	vsAccount := acc("vs_account__________")
	vs := vsAccountEntry(vsAccount)
	require.NoError(t, f.keeper.CheckVSAccountAuthorization(f.ctx, testCorpID, vsAccount, &types.AccreditationView{Id: 4, Active: true}, vs, mtIssuance))
	require.NoError(t, f.keeper.CheckVSAccountAuthorization(f.ctx, testCorpID, vsAccount, nil, vs, mtIssuance))
}

func TestCheckVSAccount_WrongCorporation(t *testing.T) {
	f := initFixture(t)
	vsAccount := acc("vs_account__________")
	err := f.keeper.CheckVSAccountAuthorization(f.ctx, testCorpID+1, vsAccount, nil, vsAccountEntry(vsAccount), mtIssuance)
	require.ErrorIs(t, err, types.ErrVSAccountNotAuthorized)
}

func TestCheckVSAccount_WrongSigner(t *testing.T) {
	f := initFixture(t)
	vs := vsAccountEntry(acc("vs_account__________"))
	err := f.keeper.CheckVSAccountAuthorization(f.ctx, testCorpID, acc("someone_else________"), nil, vs, mtIssuance)
	require.ErrorIs(t, err, types.ErrVSAccountNotAuthorized)

	vs.VsAccount = ""
	err = f.keeper.CheckVSAccountAuthorization(f.ctx, testCorpID, acc("someone_else________"), nil, vs, mtIssuance)
	require.ErrorIs(t, err, types.ErrVSAccountNotAuthorized)
}

func TestCheckVSAccount_NoMsgTypes(t *testing.T) {
	f := initFixture(t)
	vsAccount := acc("vs_account__________")
	vs := vsAccountEntry(vsAccount)
	vs.MsgTypes = nil
	err := f.keeper.CheckVSAccountAuthorization(f.ctx, testCorpID, vsAccount, nil, vs, mtIssuance)
	require.ErrorIs(t, err, types.ErrVSAccountNotAuthorized)
}

func TestCheckVSAccount_MsgTypeNotAuthorized(t *testing.T) {
	f := initFixture(t)
	vsAccount := acc("vs_account__________")
	err := f.keeper.CheckVSAccountAuthorization(f.ctx, testCorpID, vsAccount, nil, vsAccountEntry(vsAccount), types.MsgType_MSG_TYPE_STORE_DIGEST)
	require.ErrorIs(t, err, types.ErrAuthzMsgTypeNotFound)
}

// GF-CHECK-2 never applies to a VS account.
func TestCheckVSAccount_NoGFCheck2(t *testing.T) {
	f := initFixture(t)
	f.gf.err = errNotAccepted
	vsAccount := acc("vs_account__________")
	require.NoError(t, f.keeper.CheckVSAccountAuthorization(f.ctx, testCorpID, vsAccount, nil, vsAccountEntry(vsAccount), mtIssuance))
	require.Zero(t, f.gf.calls)
}

// [AUTHZ-CHECK-3] step 5: an ended cycle resets the budget at block time.
func TestCheckVSAccount_CycleReset(t *testing.T) {
	f := initFixture(t)
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	ctx := sdk.UnwrapSDKContext(f.ctx).WithBlockTime(now)
	vsAccount := acc("vs_account__________")
	period := 24 * time.Hour
	vs := vsAccountEntry(vsAccount)
	vs.Period = &period
	limit := sdk.NewCoins(sdk.NewInt64Coin("uvna", 100))
	for _, ended := range []time.Time{now, now.Add(-time.Hour)} {
		end := ended
		f.budgets[testVsID] = types.VsAccountBudget{SpendLimit: limit, RemainingSpend: sdk.NewCoins(sdk.NewInt64Coin("uvna", 5)), Expiration: &end}

		require.NoError(t, f.keeper.CheckVSAccountAuthorization(ctx, testCorpID, vsAccount, nil, vs, mtIssuance))
		got := f.budgets[testVsID]
		require.Equal(t, limit, got.RemainingSpend)
		require.Equal(t, now.Add(period), *got.Expiration)
	}
}

func TestCheckVSAccount_CycleNotDue(t *testing.T) {
	f := initFixture(t)
	now := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	ctx := sdk.UnwrapSDKContext(f.ctx).WithBlockTime(now)
	vsAccount := acc("vs_account__________")
	period := 24 * time.Hour
	vs := vsAccountEntry(vsAccount)
	vs.Period = &period
	end := now.Add(time.Second)
	want := types.VsAccountBudget{
		SpendLimit:     sdk.NewCoins(sdk.NewInt64Coin("uvna", 100)),
		RemainingSpend: sdk.NewCoins(sdk.NewInt64Coin("uvna", 5)),
		Expiration:     &end,
	}
	f.budgets[testVsID] = want

	require.NoError(t, f.keeper.CheckVSAccountAuthorization(ctx, testCorpID, vsAccount, nil, vs, mtIssuance))
	require.Equal(t, want, f.budgets[testVsID])
}

// Without a period the budget is a total: no cycle, no reset.
func TestCheckVSAccount_NoPeriodNoReset(t *testing.T) {
	f := initFixture(t)
	vsAccount := acc("vs_account__________")
	want := types.VsAccountBudget{
		SpendLimit:     sdk.NewCoins(sdk.NewInt64Coin("uvna", 100)),
		RemainingSpend: sdk.NewCoins(sdk.NewInt64Coin("uvna", 5)),
	}
	f.budgets[testVsID] = want

	require.NoError(t, f.keeper.CheckVSAccountAuthorization(f.ctx, testCorpID, vsAccount, nil, vsAccountEntry(vsAccount), mtIssuance))
	require.Equal(t, want, f.budgets[testVsID])
}

// [AUTHZ-CHECK-3] step 6: the budget, when there is one, covers and is debited
// per denom.
func TestConsumeVSAccountSpend(t *testing.T) {
	f := initFixture(t)
	uvna := func(n int64) sdk.Coins { return sdk.NewCoins(sdk.NewInt64Coin("uvna", n)) }

	// No budget: nothing to check or debit.
	require.NoError(t, f.keeper.ConsumeVSAccountSpend(f.ctx, testVsID, uvna(999)))
	require.Empty(t, f.budgets)

	f.budgets[testVsID] = types.VsAccountBudget{SpendLimit: uvna(100), RemainingSpend: uvna(100)}
	require.NoError(t, f.keeper.ConsumeVSAccountSpend(f.ctx, testVsID, uvna(60)))
	require.Equal(t, uvna(40), f.budgets[testVsID].RemainingSpend)
	require.Equal(t, uvna(100), f.budgets[testVsID].SpendLimit)

	require.ErrorIs(t, f.keeper.ConsumeVSAccountSpend(f.ctx, testVsID, uvna(41)), types.ErrAuthzSpendLimitExceeded)
	require.ErrorIs(t, f.keeper.ConsumeVSAccountSpend(f.ctx, testVsID, sdk.NewCoins(sdk.NewInt64Coin("other", 1))), types.ErrAuthzSpendLimitExceeded)
	require.Equal(t, uvna(40), f.budgets[testVsID].RemainingSpend)

	require.NoError(t, f.keeper.ConsumeVSAccountSpend(f.ctx, testVsID, sdk.NewCoins()))
	require.Equal(t, uvna(40), f.budgets[testVsID].RemainingSpend)
}

// [AUTHZ-CHECK-4]: a corporation-paid fee needs a non-empty fee_spend_limit;
// other payers are not concerned.
func TestCheckVSAccountFeeGrant(t *testing.T) {
	corporation := sdk.AccAddress([]byte("corp_policy_address_"))
	other := sdk.AccAddress([]byte("other_fee_granter___"))
	fee := sdk.NewCoins(sdk.NewInt64Coin("uvna", 10))

	cases := []struct {
		name          string
		tx            sdk.Tx // nil: no ante handler ran
		feeSpendLimit sdk.Coins
		wantErr       error
	}{
		{name: "corp pays, fee_spend_limit set", tx: feeTxStub{granter: corporation}, feeSpendLimit: fee},
		{name: "corp pays, fee_spend_limit empty", tx: feeTxStub{granter: corporation}, wantErr: types.ErrVSAccountFeeGrantNotEnabled},
		{name: "other granter", tx: feeTxStub{granter: other}},
		{name: "no granter", tx: feeTxStub{}},
		{name: "not a fee tx", tx: plainTxStub{}},
		{name: "no ante handler"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := initFixture(t)
			ctx := sdk.UnwrapSDKContext(f.ctx)
			if tc.tx != nil {
				ctx = anteCtx(t, ctx, tc.tx)
			}
			vs := vsAccountEntry(acc("vs_account__________"))
			vs.FeeSpendLimit = tc.feeSpendLimit
			err := f.keeper.CheckVSAccountFeeGrant(ctx, corporation.String(), vs)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
