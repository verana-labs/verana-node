package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/verana-labs/verana-node/x/de/keeper"
	"github.com/verana-labs/verana-node/x/de/types"
)

func TestGetSponsorship(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	s := types.Sponsorship{SponsorCorporationId: sponsorID, SponsoredCorporationId: sponsoredID, CoversFees: true, Created: time.Unix(10, 0).UTC()}
	setSponsorship(t, f, ctx, s)

	res, err := qs.GetSponsorship(ctx, &types.QueryGetSponsorshipRequest{SponsorCorporationId: sponsorID, SponsoredCorporationId: sponsoredID})
	require.NoError(t, err)
	require.Equal(t, s, res.Sponsorship)

	_, err = qs.GetSponsorship(ctx, &types.QueryGetSponsorshipRequest{SponsorCorporationId: sponsoredID, SponsoredCorporationId: sponsorID})
	require.Equal(t, codes.NotFound, status.Code(err))
	_, err = qs.GetSponsorship(ctx, &types.QueryGetSponsorshipRequest{SponsorCorporationId: sponsorID})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}

func TestListSponsorships(t *testing.T) {
	f, _, ctx := setupMsgServer(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(now)
	qs := keeper.NewQueryServerImpl(f.keeper)
	past, period := now.Add(-time.Hour), time.Hour
	// 4 elects the sponsor, 5 elects nobody, 6 elects the sponsor but its
	// sponsorship expired, 7 cycles past its expiration.
	f.corpKeeper.sponsors[4], f.corpKeeper.sponsors[6], f.corpKeeper.sponsors[7] = sponsorID, sponsorID, sponsorID
	setSponsorship(t, f, ctx, types.Sponsorship{SponsorCorporationId: sponsorID, SponsoredCorporationId: 4})
	setSponsorship(t, f, ctx, types.Sponsorship{SponsorCorporationId: sponsorID, SponsoredCorporationId: 5})
	setSponsorship(t, f, ctx, types.Sponsorship{SponsorCorporationId: sponsorID, SponsoredCorporationId: 6, Expiration: &past})
	setSponsorship(t, f, ctx, types.Sponsorship{SponsorCorporationId: sponsorID, SponsoredCorporationId: 7, Expiration: &past, Period: &period})
	setSponsorship(t, f, ctx, types.Sponsorship{SponsorCorporationId: 9, SponsoredCorporationId: 4})

	res, err := qs.ListSponsorships(ctx, &types.QueryListSponsorshipsRequest{SponsorCorporationId: sponsorID})
	require.NoError(t, err)
	got := map[uint64]bool{}
	for _, l := range res.Sponsorships {
		require.Equal(t, sponsorID, l.Sponsorship.SponsorCorporationId)
		got[l.Sponsorship.SponsoredCorporationId] = l.Active
	}
	require.Equal(t, map[uint64]bool{4: true, 5: false, 6: false, 7: true}, got)

	res, err = qs.ListSponsorships(ctx, &types.QueryListSponsorshipsRequest{SponsorCorporationId: sponsorID, ResponseMaxSize: 2})
	require.NoError(t, err)
	require.Len(t, res.Sponsorships, 2)

	_, err = qs.ListSponsorships(ctx, &types.QueryListSponsorshipsRequest{})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = qs.ListSponsorships(ctx, &types.QueryListSponsorshipsRequest{SponsorCorporationId: sponsorID, ResponseMaxSize: 1025})
	require.Equal(t, codes.InvalidArgument, status.Code(err))
}
