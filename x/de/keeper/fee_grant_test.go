package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

func slotAddrs(corpID uint64, grantee string) (sdk.AccAddress, sdk.AccAddress) {
	granter, _ := sdk.AccAddressFromBech32(corpPolicyAddr(corpID))
	granteeAddr, _ := sdk.AccAddressFromBech32(grantee)
	return granter, granteeAddr
}

// slot returns the allowance from corpID's policy address to grantee, nil if none.
func slot(f *fixture, corpID uint64, grantee string) feegrant.FeeAllowanceI {
	return f.feegrants.grants[feegrantKey(slotAddrs(corpID, grantee))]
}

// putSlot places an allowance the module did not create, replacing any.
func putSlot(f *fixture, corpID uint64, grantee string, a feegrant.FeeAllowanceI) {
	f.feegrants.grants[feegrantKey(slotAddrs(corpID, grantee))] = a
}

func hasFeeGrant(t *testing.T, f *fixture, ctx sdk.Context, corpID uint64, grantee string) bool {
	t.Helper()
	has, err := f.keeper.FeeGrants.Has(ctx, collections.Join(corpID, grantee))
	require.NoError(t, err)
	return has
}

func innerAllowance(t *testing.T, a feegrant.FeeAllowanceI) feegrant.FeeAllowanceI {
	t.Helper()
	wrapped, ok := a.(*feegrant.AllowedMsgAllowance)
	require.True(t, ok)
	inner, err := wrapped.GetAllowance()
	require.NoError(t, err)
	return inner
}

func eventAttrs(ctx sdk.Context, typ string) []map[string]string {
	var out []map[string]string
	for _, e := range ctx.EventManager().Events() {
		if e.Type != typ {
			continue
		}
		m := map[string]string{}
		for _, a := range e.Attributes {
			m[a.Key] = a.Value
		}
		out = append(out, m)
	}
	return out
}

func TestGrantFeeAllowance_Shapes(t *testing.T) {
	period := time.Hour
	for name, tc := range map[string]struct {
		limit    sdk.Coins
		period   *time.Duration
		periodic bool
	}{
		"limit and period": {uvna(10), &period, true},
		"period only":      {nil, &period, false},
		"limit only":       {uvna(10), nil, false},
	} {
		t.Run(name, func(t *testing.T) {
			f, _, ctx := setupMsgServer(t)
			grantee := acc("grantee_____________")
			exp := ctx.BlockTime().Add(2 * time.Hour)
			require.NoError(t, f.keeper.GrantFeeAllowance(ctx, 7, grantee, []types.MsgType{mtEcosystem}, &exp, tc.limit, tc.period))

			fg, err := f.keeper.FeeGrants.Get(ctx, collections.Join(uint64(7), grantee))
			require.NoError(t, err)
			require.Equal(t, tc.limit, fg.SpendLimit)
			require.Equal(t, tc.limit, fg.RemainingSpend)
			require.True(t, fg.Expiration.Equal(exp))

			inner := innerAllowance(t, slot(f, 7, grantee))
			if tc.periodic {
				require.Equal(t, &period, fg.Period)
				pa := inner.(*feegrant.PeriodicAllowance)
				require.Equal(t, period, pa.Period)
				require.Equal(t, tc.limit, pa.PeriodSpendLimit)
				require.Equal(t, tc.limit, pa.PeriodCanSpend)
				require.True(t, pa.PeriodReset.Equal(exp))
				require.Nil(t, pa.Basic.Expiration)
				return
			}
			require.Nil(t, fg.Period)
			ba := inner.(*feegrant.BasicAllowance)
			require.Equal(t, tc.limit, ba.SpendLimit)
			require.True(t, ba.Expiration.Equal(exp))
		})
	}
}

func TestGrantFeeAllowance_ReplacesOwnAllowance(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	grantee := acc("grantee_____________")
	require.NoError(t, f.keeper.GrantFeeAllowance(ctx, 7, grantee, []types.MsgType{mtEcosystem}, nil, uvna(5), nil))

	require.NoError(t, f.keeper.GrantFeeAllowance(ctx, 7, grantee, []types.MsgType{mtSchema}, nil, uvna(9), nil))

	a := slot(f, 7, grantee).(*feegrant.AllowedMsgAllowance)
	require.Equal(t, []string{"/verana.cs.v1.MsgCreateCredentialSchema"}, a.AllowedMessages)
	require.Equal(t, uvna(9), innerAllowance(t, a).(*feegrant.BasicAllowance).SpendLimit)
}

// [MOD-DE-MSG-1-2] An allowance with no FeeGrant entry is not the module's.
func TestGrantFeeAllowance_ForeignAllowance(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	grantee := acc("grantee_____________")
	foreign := &feegrant.BasicAllowance{SpendLimit: uvna(3)}
	putSlot(f, 7, grantee, foreign)

	err := f.keeper.GrantFeeAllowance(ctx, 7, grantee, []types.MsgType{mtEcosystem}, nil, uvna(10), nil)
	require.ErrorIs(t, err, types.ErrFeeAllowanceNotOwned)
	require.Same(t, foreign, slot(f, 7, grantee))
	require.False(t, hasFeeGrant(t, f, ctx, 7, grantee))

	// Callers inherit the abort; a grant without fee allowance leaves the slot alone.
	corp := acc("corp________________")
	putSlot(f, testCorpID, grantee, foreign)
	_, err = ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corp, Delegate: corp, Grantee: grantee,
		MsgTypes: []types.MsgType{mtEcosystem}, WithFeegrant: true, FeegrantSpendLimit: uvna(10),
	})
	require.ErrorIs(t, err, types.ErrFeeAllowanceNotOwned)

	_, err = ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corp, Delegate: corp, Grantee: grantee, MsgTypes: []types.MsgType{mtEcosystem},
	})
	require.NoError(t, err)
	require.Same(t, foreign, slot(f, testCorpID, grantee))
}

// [MOD-DE-MSG-1-2] An entry ended by expiry does not own a live allowance.
func TestGrantFeeAllowance_EndedEntryLiveAllowance(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	grantee := acc("grantee_____________")
	exp := ctx.BlockTime().Add(time.Hour)
	require.NoError(t, f.keeper.GrantFeeAllowance(ctx, 7, grantee, []types.MsgType{mtEcosystem}, &exp, nil, nil))

	foreign := &feegrant.BasicAllowance{}
	putSlot(f, 7, grantee, foreign)
	for name, at := range map[string]time.Time{"at expiration": exp, "after": exp.Add(time.Hour)} {
		t.Run(name, func(t *testing.T) {
			later := ctx.WithBlockTime(at)
			next := at.Add(time.Hour)
			err := f.keeper.GrantFeeAllowance(later, 7, grantee, []types.MsgType{mtEcosystem}, &next, nil, nil)
			require.ErrorIs(t, err, types.ErrFeeAllowanceNotOwned)
			require.Same(t, foreign, slot(f, 7, grantee))
		})
	}
}

// A grant whose allowance has lapsed can be granted again, whether
// x/feegrant has pruned the allowance yet or not.
func TestGrantFeeAllowance_LapsedRegrant(t *testing.T) {
	for name, pruned := range map[string]bool{"lingering": false, "pruned": true} {
		t.Run(name, func(t *testing.T) {
			f, _, ctx := setupMsgServer(t)
			grantee := acc("grantee_____________")
			exp := ctx.BlockTime().Add(time.Hour)
			require.NoError(t, f.keeper.GrantFeeAllowance(ctx, 7, grantee, []types.MsgType{mtEcosystem}, &exp, nil, nil))
			if pruned {
				delete(f.feegrants.grants, feegrantKey(slotAddrs(7, grantee)))
			}

			later := ctx.WithBlockTime(exp.Add(time.Hour))
			next := exp.Add(2 * time.Hour)
			require.NoError(t, f.keeper.GrantFeeAllowance(later, 7, grantee, []types.MsgType{mtEcosystem}, &next, nil, nil))
			require.True(t, innerAllowance(t, slot(f, 7, grantee)).(*feegrant.BasicAllowance).Expiration.Equal(next))
		})
	}
}

// A periodic entry auto-renews, so it owns whatever sits in its slot.
func TestGrantFeeAllowance_PeriodicEntryOwnsSlot(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	grantee := acc("grantee_____________")
	period := time.Hour
	exp := ctx.BlockTime().Add(period)
	require.NoError(t, f.keeper.GrantFeeAllowance(ctx, 7, grantee, []types.MsgType{mtEcosystem}, &exp, uvna(10), &period))

	later := ctx.WithBlockTime(exp.Add(3 * period))
	next := exp.Add(4 * period)
	require.NoError(t, f.keeper.GrantFeeAllowance(later, 7, grantee, []types.MsgType{mtEcosystem}, &next, uvna(20), &period))
	require.Equal(t, uvna(20), innerAllowance(t, slot(f, 7, grantee)).(*feegrant.PeriodicAllowance).PeriodCanSpend)
}

// [MOD-DE-MSG-2-4] With no entry, no allowance is touched.
func TestRevokeFeeAllowance_NoEntryLeavesAllowance(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	grantee := acc("grantee_____________")
	foreign := &feegrant.BasicAllowance{}
	putSlot(f, 7, grantee, foreign)

	require.NoError(t, f.keeper.RevokeFeeAllowance(ctx, 7, grantee))
	require.Same(t, foreign, slot(f, 7, grantee))
	require.Empty(t, eventAttrs(ctx, types.EventTypeRevokeFeeAllowance))
}

func TestRevokeFeeAllowance_EndedEntry(t *testing.T) {
	for name, tc := range map[string]struct {
		foreign feegrant.FeeAllowanceI
		revoked bool
	}{
		"own lapsed allowance": {nil, true},
		"live foreign":         {&feegrant.BasicAllowance{}, false},
	} {
		t.Run(name, func(t *testing.T) {
			f, _, ctx := setupMsgServer(t)
			grantee := acc("grantee_____________")
			exp := ctx.BlockTime().Add(time.Hour)
			require.NoError(t, f.keeper.GrantFeeAllowance(ctx, 7, grantee, []types.MsgType{mtEcosystem}, &exp, nil, nil))
			if tc.foreign != nil {
				putSlot(f, 7, grantee, tc.foreign)
			}

			require.NoError(t, f.keeper.RevokeFeeAllowance(ctx.WithBlockTime(exp), 7, grantee))
			require.False(t, hasFeeGrant(t, f, ctx, 7, grantee))
			require.Equal(t, tc.revoked, slot(f, 7, grantee) == nil)
		})
	}
}

func TestRevokeFeeAllowance_EntryWithoutAllowance(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	grantee := acc("grantee_____________")
	require.NoError(t, f.keeper.GrantFeeAllowance(ctx, 7, grantee, []types.MsgType{mtEcosystem}, nil, uvna(5), nil))
	delete(f.feegrants.grants, feegrantKey(slotAddrs(7, grantee)))

	require.NoError(t, f.keeper.RevokeFeeAllowance(ctx, 7, grantee))
	require.False(t, hasFeeGrant(t, f, ctx, 7, grantee))
}

func TestFeeAllowance_Events(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	grantee := acc("grantee_____________")
	want := map[string]string{"module": types.ModuleName, "grantor_corporation_id": "7", "grantee": grantee}

	require.NoError(t, f.keeper.GrantFeeAllowance(ctx, 7, grantee, []types.MsgType{mtEcosystem}, nil, nil, nil))
	require.Equal(t, []map[string]string{want}, eventAttrs(ctx, types.EventTypeGrantFeeAllowance))

	require.NoError(t, f.keeper.RevokeFeeAllowance(ctx, 7, grantee))
	require.Equal(t, []map[string]string{want}, eventAttrs(ctx, types.EventTypeRevokeFeeAllowance))
}
