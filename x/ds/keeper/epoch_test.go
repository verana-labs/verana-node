package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/ds/types"
)

func vote(cons sdk.ConsAddress, flag cmtproto.BlockIDFlag) abci.VoteInfo {
	return abci.VoteInfo{Validator: abci.Validator{Address: cons, Power: 1}, BlockIdFlag: flag}
}

func TestCountBlock(t *testing.T) {
	f := initFixture(t)
	signer, signerCons := f.addValidator(t, "signer", 10)
	absent, absentCons := f.addValidator(t, "absent", 10)
	nilVoter, nilCons := f.addValidator(t, "nil", 10)
	jailed, _ := f.addValidator(t, "jailed", 10)
	f.staking.validators[3].Jailed = true
	removed, removedCons := f.addValidator(t, "removed", 0)

	ctx := f.ctx.WithVoteInfos([]abci.VoteInfo{
		vote(signerCons, cmtproto.BlockIDFlagCommit),
		vote(absentCons, cmtproto.BlockIDFlagAbsent),
		vote(nilCons, cmtproto.BlockIDFlagNil),
		vote(removedCons, cmtproto.BlockIDFlagCommit),
	})
	require.NoError(t, f.keeper.CountBlock(ctx))
	require.NoError(t, f.keeper.CountBlock(ctx.WithVoteInfos(nil)))

	for _, tc := range []struct {
		val            sdk.ValAddress
		signed, member uint64
	}{
		{signer, 1, 2},
		{absent, 0, 2},
		{nilVoter, 1, 2},
		{jailed, 0, 2},
	} {
		got, err := f.keeper.ValidatorBlocks.Get(ctx, tc.val)
		require.NoError(t, err)
		require.Equal(t, types.ValidatorBlocks{SignedBlocks: tc.signed, MemberBlocks: tc.member}, got)
	}
	has, err := f.keeper.ValidatorBlocks.Has(ctx, removed)
	require.NoError(t, err)
	require.False(t, has)
	n, err := f.keeper.EpochBlocks.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(2), n)
}

// budgets in minor units; the mock rate converts 1 minor unit to 2 uvna.
func setBudgets(t *testing.T, f *fixture, edit func(p *types.Params)) types.Params {
	t.Helper()
	p := types.DefaultParams()
	p.ValidatorNodeBudget = math.LegacyNewDec(1000)
	p.CouncilBudget = math.LegacyNewDec(300)
	p.FoundationBudget = math.LegacyNewDec(200)
	if edit != nil {
		edit(&p)
	}
	require.NoError(t, f.keeper.Params.Set(f.ctx, p))
	return p
}

func setBlocks(t *testing.T, f *fixture, epochBlocks uint64, counters map[string]types.ValidatorBlocks) {
	t.Helper()
	for val, b := range counters {
		addr, err := sdk.ValAddressFromBech32(val)
		require.NoError(t, err)
		require.NoError(t, f.keeper.ValidatorBlocks.Set(f.ctx, addr, b))
	}
	require.NoError(t, f.keeper.EpochBlocks.Set(f.ctx, epochBlocks))
}

func closeEpochAttrs(t *testing.T, ctx sdk.Context) []abci.EventAttribute {
	t.Helper()
	var out []abci.EventAttribute
	for _, ev := range ctx.EventManager().ABCIEvents() {
		if ev.Type == types.EventTypeCloseEpoch {
			require.Nil(t, out, "one close_epoch event")
			out = ev.Attributes
		}
	}
	require.NotNil(t, out)
	return out
}

func attrValues(attrs []abci.EventAttribute, key string) []string {
	var out []string
	for _, a := range attrs {
		if a.Key == key {
			out = append(out, a.Value)
		}
	}
	return out
}

func requireAmount(t *testing.T, want int64, got math.Int) {
	t.Helper()
	require.Equal(t, math.NewInt(want).String(), got.String())
}

func requireCountersReset(t *testing.T, f *fixture) {
	t.Helper()
	empty, err := f.keeper.ValidatorBlocks.Iterate(f.ctx, nil)
	require.NoError(t, err)
	defer empty.Close()
	require.False(t, empty.Valid())
	has, err := f.keeper.EpochBlocks.Has(f.ctx)
	require.NoError(t, err)
	require.False(t, has)
}

func TestPayoutEpoch(t *testing.T) {
	f := initFixture(t)
	p := setBudgets(t, f, nil)
	full, _ := f.addValidator(t, "full", 10)
	partial, _ := f.addValidator(t, "partial", 10)
	low, _ := f.addValidator(t, "low", 10)
	joined, _ := f.addValidator(t, "joined", 10)
	setBlocks(t, f, 100, map[string]types.ValidatorBlocks{
		full.String():    {SignedBlocks: 100, MemberBlocks: 100},
		partial.String(): {SignedBlocks: 95, MemberBlocks: 100},
		low.String():     {SignedBlocks: 89, MemberBlocks: 100},
		joined.String():  {SignedBlocks: 50, MemberBlocks: 50},
	})
	f.fundPool(10_000)

	require.NoError(t, f.keeper.PayoutEpoch(f.ctx, 7))

	foundation := sdk.MustAccAddressFromBech32(p.FoundationAccount)
	requireAmount(t, 2000, f.balance(sdk.AccAddress(full)))
	requireAmount(t, 1111, f.balance(sdk.AccAddress(partial)))
	requireAmount(t, 0, f.balance(sdk.AccAddress(low)))
	requireAmount(t, 1000, f.balance(sdk.AccAddress(joined)))
	requireAmount(t, 600, f.balance(f.keeper.GetAuthority()))
	requireAmount(t, 400, f.balance(foundation))
	requireAmount(t, 4889, f.bank.burned)
	requireAmount(t, 0, f.balance(poolAddr))
	requireCountersReset(t, f)

	attrs := closeEpochAttrs(t, f.ctx)
	require.Equal(t, []string{types.ModuleName}, attrValues(attrs, sdk.AttributeKeyModule))
	require.Equal(t, []string{"7"}, attrValues(attrs, types.AttributeKeyEpoch))
	require.Equal(t, []string{"100"}, attrValues(attrs, types.AttributeKeyEpochBlocks))
	require.Equal(t, []string{"false"}, attrValues(attrs, types.AttributeKeySkipped))
	require.Len(t, attrValues(attrs, types.AttributeKeyValidator), 4)
	require.ElementsMatch(t, []string{"2000", "1111", "0", "1000"}, attrValues(attrs, types.AttributeKeyValidatorPayout))
	require.Equal(t, []string{"600"}, attrValues(attrs, types.AttributeKeyCouncilPayout))
	require.Equal(t, []string{"400"}, attrValues(attrs, types.AttributeKeyFoundationPayout))
	require.Equal(t, []string{"4889"}, attrValues(attrs, types.AttributeKeyBurned))
}

func TestPayoutEpoch_UptimeFactor(t *testing.T) {
	dec := math.LegacyMustNewDecFromStr
	tests := []struct {
		name              string
		signed, member    uint64
		minUptime, target string
		want              int64
	}{
		{"at target", 99, 100, "0.90", "0.99", 2000},
		{"between", 95, 100, "0.90", "0.99", 1111},
		{"at min", 90, 100, "0.90", "0.99", 0},
		{"below min", 89, 100, "0.90", "0.99", 0},
		{"half epoch, between", 48, 50, "0.90", "0.99", 666},
		{"equal thresholds, at", 95, 100, "0.95", "0.95", 2000},
		{"equal thresholds, below", 94, 100, "0.95", "0.95", 0},
		{"zero thresholds", 0, 100, "0", "0", 2000},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := initFixture(t)
			setBudgets(t, f, func(p *types.Params) {
				p.ValidatorMinUptime, p.ValidatorTargetUptime = dec(tc.minUptime), dec(tc.target)
			})
			val, _ := f.addValidator(t, "val", 10)
			setBlocks(t, f, 100, map[string]types.ValidatorBlocks{val.String(): {SignedBlocks: tc.signed, MemberBlocks: tc.member}})
			f.fundPool(10_000)

			require.NoError(t, f.keeper.PayoutEpoch(f.ctx, 0))
			requireAmount(t, tc.want, f.balance(sdk.AccAddress(val)))
		})
	}
}

func TestPayoutEpoch_Skipped(t *testing.T) {
	tests := []struct {
		name        string
		epochBlocks uint64
		noRate      bool
	}{
		{"no block", 0, false},
		{"no valid rate", 100, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := initFixture(t)
			setBudgets(t, f, nil)
			val, _ := f.addValidator(t, "val", 10)
			if tc.epochBlocks > 0 {
				setBlocks(t, f, tc.epochBlocks, map[string]types.ValidatorBlocks{val.String(): {SignedBlocks: 100, MemberBlocks: 100}})
			}
			if tc.noRate {
				f.xr.rate = nil
			}
			f.fundPool(10_000)

			require.NoError(t, f.keeper.PayoutEpoch(f.ctx, 3))
			requireAmount(t, 10_000, f.balance(poolAddr))
			requireAmount(t, 0, f.bank.burned)
			requireAmount(t, 0, f.balance(sdk.AccAddress(val)))
			requireCountersReset(t, f)

			attrs := closeEpochAttrs(t, f.ctx)
			require.Equal(t, []string{"3"}, attrValues(attrs, types.AttributeKeyEpoch))
			require.Equal(t, []string{"true"}, attrValues(attrs, types.AttributeKeySkipped))
			require.Empty(t, attrValues(attrs, types.AttributeKeyBurned))
		})
	}
}

func TestPayoutEpoch_Shortfall(t *testing.T) {
	tests := []struct {
		name                                      string
		pool                                      int64
		full, joined, council, foundation, burned int64
	}{
		{"validators pro-rated", 1500, 1000, 500, 0, 0, 0},
		{"council partly", 3300, 2000, 1000, 300, 0, 0},
		{"foundation partly", 3700, 2000, 1000, 600, 100, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := initFixture(t)
			p := setBudgets(t, f, nil)
			full, _ := f.addValidator(t, "full", 10)
			joined, _ := f.addValidator(t, "joined", 10)
			setBlocks(t, f, 100, map[string]types.ValidatorBlocks{
				full.String():   {SignedBlocks: 100, MemberBlocks: 100},
				joined.String(): {SignedBlocks: 50, MemberBlocks: 50},
			})
			f.fundPool(tc.pool)

			require.NoError(t, f.keeper.PayoutEpoch(f.ctx, 1))
			requireAmount(t, tc.full, f.balance(sdk.AccAddress(full)))
			requireAmount(t, tc.joined, f.balance(sdk.AccAddress(joined)))
			requireAmount(t, tc.council, f.balance(f.keeper.GetAuthority()))
			requireAmount(t, tc.foundation, f.balance(sdk.MustAccAddressFromBech32(p.FoundationAccount)))
			requireAmount(t, tc.burned, f.bank.burned)
			requireAmount(t, 0, f.balance(poolAddr))
		})
	}
}

func TestPayoutEpoch_BondingEnabledBurnsRemainder(t *testing.T) {
	f := initFixture(t)
	setBudgets(t, f, func(p *types.Params) {
		p.BondingEnabled = true
		p.VnaHolderMaxYieldRate = math.LegacyMustNewDecFromStr("0.1")
	})
	setBlocks(t, f, 100, nil)
	f.fundPool(5000)

	require.NoError(t, f.keeper.PayoutEpoch(f.ctx, 1))
	requireAmount(t, 4000, f.bank.burned)
}
