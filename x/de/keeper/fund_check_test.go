package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

const (
	sponsoredID uint64 = 2
	sponsorID   uint64 = 3
)

func setSponsorship(t *testing.T, f *fixture, ctx sdk.Context, s types.Sponsorship) {
	t.Helper()
	require.NoError(t, f.keeper.Sponsorships.Set(ctx, collections.Join(s.SponsorCorporationId, s.SponsoredCorporationId), s))
}

func getSponsorship(t *testing.T, f *fixture, ctx sdk.Context) types.Sponsorship {
	t.Helper()
	s, err := f.keeper.Sponsorships.Get(ctx, collections.Join(sponsorID, sponsoredID))
	require.NoError(t, err)
	return s
}

func fund(f *fixture, id uint64, coins sdk.Coins) {
	f.bank[corpPolicyAddr(id)] = coins
}

func sponsorFailures(ctx sdk.Context) int {
	n := 0
	for _, e := range ctx.EventManager().Events() {
		if e.Type == types.EventTypeSponsorCouldNotPay {
			n++
		}
	}
	return n
}

func TestResolveDebitAccount(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	totals := uvna(30)

	cases := []struct {
		name        string
		elect       uint64
		sponsorship *types.Sponsorship
		sponsorBal  sdk.Coins
		ownBal      sdk.Coins
		totals      sdk.Coins
		wantPayer   uint64
		wantFailEvt int
		wantErr     error
	}{
		{name: "no sponsor pays own", ownBal: uvna(30), wantPayer: sponsoredID},
		{name: "no sponsor own short", ownBal: uvna(29), wantErr: types.ErrNoFundingAccount},
		{name: "election without sponsorship", elect: sponsorID, ownBal: uvna(30), wantPayer: sponsoredID},
		{
			name: "sponsor pays", elect: sponsorID,
			sponsorship: &types.Sponsorship{SpendLimit: uvna(100), RemainingSpend: uvna(100)},
			sponsorBal:  uvna(30), wantPayer: sponsorID,
		},
		{
			name: "sponsor pays without limit", elect: sponsorID,
			sponsorship: &types.Sponsorship{}, sponsorBal: uvna(30), wantPayer: sponsorID,
		},
		{
			name: "sponsor covers listed msg type", elect: sponsorID,
			sponsorship: &types.Sponsorship{MsgTypes: []types.MsgType{mtEcosystem}},
			sponsorBal:  uvna(30), wantPayer: sponsorID,
		},
		{
			name: "msg type not covered", elect: sponsorID,
			sponsorship: &types.Sponsorship{MsgTypes: []types.MsgType{mtSchema}},
			sponsorBal:  uvna(30), ownBal: uvna(30), wantPayer: sponsoredID, wantFailEvt: 1,
		},
		{
			name: "remaining short", elect: sponsorID,
			sponsorship: &types.Sponsorship{SpendLimit: uvna(100), RemainingSpend: uvna(29)},
			sponsorBal:  uvna(30), ownBal: uvna(30), wantPayer: sponsoredID, wantFailEvt: 1,
		},
		{
			name: "denom absent from limit", elect: sponsorID,
			sponsorship: &types.Sponsorship{SpendLimit: uvna(100), RemainingSpend: uvna(100)},
			sponsorBal:  uvna(30).Add(sdk.NewInt64Coin("uatom", 5)), ownBal: uvna(30).Add(sdk.NewInt64Coin("uatom", 5)),
			totals: uvna(30).Add(sdk.NewInt64Coin("uatom", 5)), wantPayer: sponsoredID, wantFailEvt: 1,
		},
		{
			name: "sponsor balance short", elect: sponsorID,
			sponsorship: &types.Sponsorship{}, sponsorBal: uvna(29), ownBal: uvna(30), wantPayer: sponsoredID, wantFailEvt: 1,
		},
		{
			name: "both short", elect: sponsorID,
			sponsorship: &types.Sponsorship{}, sponsorBal: uvna(29), ownBal: uvna(29), wantErr: types.ErrNoFundingAccount,
		},
		{
			name: "expired sponsorship", elect: sponsorID,
			sponsorship: &types.Sponsorship{Expiration: &past}, sponsorBal: uvna(30), ownBal: uvna(30), wantPayer: sponsoredID,
		},
		{
			name: "unexpired sponsorship", elect: sponsorID,
			sponsorship: &types.Sponsorship{Expiration: &future}, sponsorBal: uvna(30), wantPayer: sponsorID,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, _, ctx := setupMsgServer(t)
			ctx = ctx.WithBlockTime(now).WithEventManager(sdk.NewEventManager())
			if tc.elect != 0 {
				f.corpKeeper.sponsors[sponsoredID] = tc.elect
			}
			if tc.sponsorship != nil {
				s := *tc.sponsorship
				s.SponsorCorporationId, s.SponsoredCorporationId = sponsorID, sponsoredID
				setSponsorship(t, f, ctx, s)
			}
			fund(f, sponsorID, tc.sponsorBal)
			fund(f, sponsoredID, tc.ownBal)
			want := totals
			if tc.totals != nil {
				want = tc.totals
			}

			payer, err := f.keeper.ResolveDebitAccount(ctx, sponsoredID, mtEcosystem, want)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, corpPolicyAddr(tc.wantPayer), payer.String())
			require.Equal(t, tc.wantFailEvt, sponsorFailures(ctx))
		})
	}
}

func TestResolveDebitAccount_DecrementsRemaining(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	f.corpKeeper.sponsors[sponsoredID] = sponsorID
	setSponsorship(t, f, ctx, types.Sponsorship{
		SponsorCorporationId: sponsorID, SponsoredCorporationId: sponsoredID,
		SpendLimit: uvna(100), RemainingSpend: uvna(100),
	})
	fund(f, sponsorID, uvna(1000))

	_, err := f.keeper.ResolveDebitAccount(ctx, sponsoredID, mtEcosystem, uvna(30))
	require.NoError(t, err)
	require.Equal(t, uvna(70), getSponsorship(t, f, ctx).RemainingSpend)

	_, err = f.keeper.ResolveDebitAccount(ctx, sponsoredID, mtEcosystem, uvna(70))
	require.NoError(t, err)
	require.True(t, getSponsorship(t, f, ctx).RemainingSpend.IsZero())
}

func TestResolveDebitAccount_CycleReset(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(now)
	ended, period := now.Add(-time.Minute), 24*time.Hour
	f.corpKeeper.sponsors[sponsoredID] = sponsorID
	setSponsorship(t, f, ctx, types.Sponsorship{
		SponsorCorporationId: sponsorID, SponsoredCorporationId: sponsoredID,
		SpendLimit: uvna(100), Period: &period, Expiration: &ended,
	})
	fund(f, sponsorID, uvna(1000))

	payer, err := f.keeper.ResolveDebitAccount(ctx, sponsoredID, mtEcosystem, uvna(30))
	require.NoError(t, err)
	require.Equal(t, corpPolicyAddr(sponsorID), payer.String())
	s := getSponsorship(t, f, ctx)
	require.Equal(t, uvna(70), s.RemainingSpend)
	require.Equal(t, now.Add(period), *s.Expiration)
}

func TestResolveCreditAccount(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	cases := []struct {
		name        string
		elect       uint64
		sponsorship *types.Sponsorship
		want        uint64
	}{
		{name: "no sponsor", want: sponsoredID},
		{name: "active sponsor", elect: sponsorID, sponsorship: &types.Sponsorship{MsgTypes: []types.MsgType{mtSchema}}, want: sponsorID},
		{name: "expired sponsorship", elect: sponsorID, sponsorship: &types.Sponsorship{Expiration: &past}, want: sponsoredID},
		{name: "election without sponsorship", elect: sponsorID, want: sponsoredID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, _, ctx := setupMsgServer(t)
			ctx = ctx.WithBlockTime(now)
			if tc.elect != 0 {
				f.corpKeeper.sponsors[sponsoredID] = tc.elect
			}
			if tc.sponsorship != nil {
				s := *tc.sponsorship
				s.SponsorCorporationId, s.SponsoredCorporationId = sponsorID, sponsoredID
				setSponsorship(t, f, ctx, s)
			}
			got, err := f.keeper.ResolveCreditAccount(ctx, sponsoredID)
			require.NoError(t, err)
			require.Equal(t, corpPolicyAddr(tc.want), got.String())
		})
	}
}

func TestFeeSponsor(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name        string
		granter     string
		sponsorship *types.Sponsorship
		want        string
	}{
		{name: "covers fees", granter: corpPolicyAddr(sponsoredID), sponsorship: &types.Sponsorship{CoversFees: true}, want: corpPolicyAddr(sponsorID)},
		{name: "does not cover fees", granter: corpPolicyAddr(sponsoredID), sponsorship: &types.Sponsorship{}},
		{name: "no sponsorship", granter: corpPolicyAddr(sponsoredID)},
		{name: "granter not a corporation", granter: acc("plain_granter_______"), sponsorship: &types.Sponsorship{CoversFees: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, _, ctx := setupMsgServer(t)
			ctx = ctx.WithBlockTime(now)
			f.corpKeeper.sponsors[sponsoredID] = sponsorID
			if tc.sponsorship != nil {
				s := *tc.sponsorship
				s.SponsorCorporationId, s.SponsoredCorporationId = sponsorID, sponsoredID
				setSponsorship(t, f, ctx, s)
			}
			got, err := f.keeper.FeeSponsor(ctx, sdk.MustAccAddressFromBech32(tc.granter))
			require.NoError(t, err)
			if tc.want == "" {
				require.Nil(t, got)
				return
			}
			require.Equal(t, tc.want, got.String())
		})
	}
}
