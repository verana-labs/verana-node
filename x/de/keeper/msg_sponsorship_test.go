package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	feegrant "github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/keeper"
	"github.com/verana-labs/verana-node/x/de/types"
)

const mtRevokeSponsorship = types.MsgType_MSG_TYPE_REVOKE_SPONSORSHIP

// The sponsor is testCorpID, the corporation every policy address resolves to.
var sponsorPolicy = corpPolicyAddr(testCorpID)

func lastEvent(ctx sdk.Context, typ string) map[string]string {
	var out map[string]string
	for _, e := range ctx.EventManager().Events() {
		if e.Type != typ {
			continue
		}
		out = map[string]string{}
		for _, a := range e.Attributes {
			out[a.Key] = a.Value
		}
	}
	return out
}

func sponsorAllowance(t *testing.T, f *fixture, ctx sdk.Context) (feegrant.FeeAllowanceI, error) {
	t.Helper()
	return f.feegrants.GetAllowance(ctx, sdk.MustAccAddressFromBech32(sponsorPolicy), sdk.MustAccAddressFromBech32(corpPolicyAddr(sponsoredID)))
}

func grantMsg(mut func(*types.MsgGrantSponsorship)) *types.MsgGrantSponsorship {
	m := &types.MsgGrantSponsorship{Signer: sponsorPolicy, SponsoredCorporationId: sponsoredID}
	if mut != nil {
		mut(m)
	}
	return m
}

func TestGrantSponsorship_Create(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(now)
	exp, period := now.Add(time.Hour), 24*time.Hour

	_, err := ms.GrantSponsorship(ctx, grantMsg(func(m *types.MsgGrantSponsorship) {
		m.MsgTypes = []types.MsgType{mtEcosystem}
		m.SpendLimit, m.Period, m.Expiration = uvna(100), &period, &exp
		m.FeeSpendLimit = uvna(5)
	}))
	require.NoError(t, err)

	s, err := f.keeper.Sponsorships.Get(ctx, collections.Join(testCorpID, sponsoredID))
	require.NoError(t, err)
	require.Equal(t, types.Sponsorship{
		SponsorCorporationId: testCorpID, SponsoredCorporationId: sponsoredID,
		MsgTypes: []types.MsgType{mtEcosystem}, SpendLimit: uvna(100), RemainingSpend: uvna(100),
		Expiration: &exp, Period: &period, Created: now,
	}, s)
	_, err = sponsorAllowance(t, f, ctx)
	require.Error(t, err)

	ev := lastEvent(ctx, types.EventTypeGrantSponsorship)
	require.Equal(t, "1", ev[types.AttributeKeySponsorCorporationID])
	require.Equal(t, "2", ev[types.AttributeKeySponsoredCorporationID])
	require.Equal(t, "2026-01-01T00:00:00Z", ev["now"])
	require.Contains(t, ev["msg"], `"sponsored_corporation_id":"2"`)
}

func TestGrantSponsorship_Corporations(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	f.corpKeeper.unregistered[sponsorPolicy] = true
	_, err := ms.GrantSponsorship(ctx, grantMsg(nil))
	require.Error(t, err)

	f, ms, ctx = setupMsgServer(t)
	_, err = ms.GrantSponsorship(ctx, grantMsg(func(m *types.MsgGrantSponsorship) { m.SponsoredCorporationId = testCorpID }))
	require.ErrorIs(t, err, types.ErrInvalidSponsorship)

	f.corpKeeper.missing[sponsoredID] = true
	_, err = ms.GrantSponsorship(ctx, grantMsg(nil))
	require.ErrorIs(t, err, types.ErrInvalidSponsorship)
}

func TestGrantSponsorship_InvalidTerms(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	past, future, period, zero := now.Add(-time.Second), now.Add(time.Hour), time.Hour, time.Duration(0)
	invalid := sdk.Coins{sdk.Coin{Denom: "uvna", Amount: sdkmath.ZeroInt()}}
	cases := []struct {
		name string
		mut  func(*types.MsgGrantSponsorship)
		want error
	}{
		{"msg type outside the sponsorship list", func(m *types.MsgGrantSponsorship) { m.MsgTypes = []types.MsgType{mtSchema} }, types.ErrInvalidMsgType},
		{"invalid spend limit", func(m *types.MsgGrantSponsorship) { m.SpendLimit = invalid }, types.ErrInvalidSpendLimit},
		{"invalid fee spend limit", func(m *types.MsgGrantSponsorship) { m.FeeSpendLimit = invalid }, types.ErrInvalidSpendLimit},
		{"zero period", func(m *types.MsgGrantSponsorship) {
			m.Period, m.SpendLimit, m.Expiration = &zero, uvna(1), &future
		}, types.ErrInvalidSponsorship},
		{"period without spend limit", func(m *types.MsgGrantSponsorship) { m.Period, m.Expiration = &period, &future }, types.ErrInvalidSponsorship},
		{"period without expiration", func(m *types.MsgGrantSponsorship) { m.Period, m.SpendLimit = &period, uvna(1) }, types.ErrInvalidSponsorship},
		{"expiration in the past", func(m *types.MsgGrantSponsorship) { m.Expiration = &past }, types.ErrExpirationInPast},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, ms, ctx := setupMsgServer(t)
			_, err := ms.GrantSponsorship(ctx.WithBlockTime(now), grantMsg(tc.mut))
			require.ErrorIs(t, err, tc.want)
		})
	}
}

func TestGrantSponsorship_SlotTaken(t *testing.T) {
	sponsored := corpPolicyAddr(sponsoredID)
	plainAllowance := func(t *testing.T, f *fixture, ctx sdk.Context) {
		require.NoError(t, f.feegrants.GrantAllowance(ctx, sdk.MustAccAddressFromBech32(sponsorPolicy), sdk.MustAccAddressFromBech32(sponsored), &feegrant.BasicAllowance{}))
	}
	cases := []struct {
		name  string
		setup func(t *testing.T, f *fixture, ctx sdk.Context)
		mut   func(*types.MsgGrantSponsorship)
	}{
		{"fee grant of the sponsor", func(t *testing.T, f *fixture, ctx sdk.Context) {
			require.NoError(t, f.keeper.FeeGrants.Set(ctx, collections.Join(testCorpID, sponsored), types.FeeGrant{}))
		}, func(m *types.MsgGrantSponsorship) { m.CoversFees = true }},
		{"VS account of the sponsor", func(t *testing.T, f *fixture, ctx sdk.Context) {
			require.NoError(t, f.keeper.VSOAByCorpOp.Set(ctx, collections.Join(testCorpID, sponsored), 4))
		}, nil},
		{"allowance with no sponsorship", plainAllowance, nil},
		{"allowance with a sponsorship that pays no fees", func(t *testing.T, f *fixture, ctx sdk.Context) {
			setSponsorship(t, f, ctx, types.Sponsorship{SponsorCorporationId: testCorpID, SponsoredCorporationId: sponsoredID})
			plainAllowance(t, f, ctx)
		}, func(m *types.MsgGrantSponsorship) { m.CoversFees = true }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, ms, ctx := setupMsgServer(t)
			tc.setup(t, f, ctx)
			_, err := ms.GrantSponsorship(ctx, grantMsg(tc.mut))
			require.ErrorIs(t, err, types.ErrSponsorshipSlotTaken)
		})
	}
}

func TestGrantSponsorship_FeeAllowance(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	exp, period := now.Add(time.Hour), 24*time.Hour
	cases := []struct {
		name string
		mut  func(*types.MsgGrantSponsorship)
		want feegrant.FeeAllowanceI
	}{
		{"periodic", func(m *types.MsgGrantSponsorship) {
			m.SpendLimit, m.Period, m.Expiration, m.FeeSpendLimit = uvna(100), &period, &exp, uvna(5)
		}, &feegrant.PeriodicAllowance{Period: period, PeriodSpendLimit: uvna(5), PeriodCanSpend: uvna(5), PeriodReset: exp}},
		{"basic with limit and expiration", func(m *types.MsgGrantSponsorship) {
			m.Expiration, m.FeeSpendLimit = &exp, uvna(5)
		}, &feegrant.BasicAllowance{SpendLimit: uvna(5), Expiration: &exp}},
		{"unlimited", nil, &feegrant.BasicAllowance{}},
		{"cycling without fee limit", func(m *types.MsgGrantSponsorship) {
			m.SpendLimit, m.Period, m.Expiration = uvna(100), &period, &exp
		}, &feegrant.BasicAllowance{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, ms, ctx := setupMsgServer(t)
			ctx = ctx.WithBlockTime(now)
			_, err := ms.GrantSponsorship(ctx, grantMsg(func(m *types.MsgGrantSponsorship) {
				m.CoversFees = true
				if tc.mut != nil {
					tc.mut(m)
				}
			}))
			require.NoError(t, err)

			a, err := sponsorAllowance(t, f, ctx)
			require.NoError(t, err)
			allowed := a.(*feegrant.AllowedMsgAllowance)
			require.Equal(t, testMaxGasURLs, allowed.AllowedMessages)
			inner, err := allowed.GetAllowance()
			require.NoError(t, err)
			require.Equal(t, tc.want, inner)
		})
	}
}

func TestGrantSponsorship_FeeAllowanceWithoutMaxGas(t *testing.T) {
	for _, covers := range []bool{true, false} {
		f, ms, ctx := setupMsgServer(t)
		f.keeper.SetNetworkFeeKeeper(keeper.StubNetworkFeeKeeper{})
		_, err := ms.GrantSponsorship(ctx, grantMsg(func(m *types.MsgGrantSponsorship) { m.CoversFees = covers }))
		if covers {
			require.ErrorContains(t, err, "max_gas_per_msg_type")
		} else {
			require.NoError(t, err)
		}
	}
}

// x/feegrant rejects an allowance whose message filter is empty.
func TestGrantSponsorship_FeeAllowanceEmptyMaxGas(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	f.keeper.SetNetworkFeeKeeper(stubNetworkFee(nil))
	_, err := ms.GrantSponsorship(ctx, grantMsg(func(m *types.MsgGrantSponsorship) { m.CoversFees = true }))
	require.ErrorIs(t, err, feegrant.ErrNoMessages)
}

func TestGrantSponsorship_RaiseOnly(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	exp, later, period, other := now.Add(time.Hour), now.Add(2*time.Hour), time.Hour, 2*time.Hour
	current := types.Sponsorship{
		SponsorCorporationId: testCorpID, SponsoredCorporationId: sponsoredID,
		MsgTypes: []types.MsgType{mtEcosystem, mtIssuance}, SpendLimit: uvna(100), RemainingSpend: uvna(40),
		Period: &period, Expiration: &exp, CoversFees: true, FeeSpendLimit: uvna(10),
	}
	raise := func(m *types.MsgGrantSponsorship) {
		m.MsgTypes, m.SpendLimit, m.Period, m.Expiration = []types.MsgType{mtEcosystem, mtIssuance}, uvna(100), &period, &later
		m.CoversFees, m.FeeSpendLimit = true, uvna(10)
	}
	cases := []struct {
		name    string
		current func(*types.Sponsorship)
		mut     func(*types.MsgGrantSponsorship)
	}{
		{"period changed", nil, func(m *types.MsgGrantSponsorship) { m.Period = &other }},
		{"period dropped", nil, func(m *types.MsgGrantSponsorship) { m.Period = nil }},
		{"msg types narrowed", nil, func(m *types.MsgGrantSponsorship) { m.MsgTypes = []types.MsgType{mtEcosystem} }},
		{"msg types listed after all", func(s *types.Sponsorship) { s.MsgTypes = nil }, nil},
		{"spend limit lowered", nil, func(m *types.MsgGrantSponsorship) { m.SpendLimit = uvna(99) }},
		{"spend limit set after none", func(s *types.Sponsorship) { s.SpendLimit, s.RemainingSpend, s.Period = nil, nil, nil }, func(m *types.MsgGrantSponsorship) { m.Period = nil }},
		{"expiration not later", nil, func(m *types.MsgGrantSponsorship) { m.Expiration = &exp }},
		{"expiration set after none", func(s *types.Sponsorship) { s.Expiration, s.Period = nil, nil }, func(m *types.MsgGrantSponsorship) {
			m.Period, m.SpendLimit = nil, nil
		}},
		{"fees no longer covered", nil, func(m *types.MsgGrantSponsorship) { m.CoversFees, m.FeeSpendLimit = false, nil }},
		{"fee limit lowered", nil, func(m *types.MsgGrantSponsorship) { m.FeeSpendLimit = uvna(9) }},
		{"fee limit set after none", func(s *types.Sponsorship) { s.FeeSpendLimit = nil }, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f, ms, ctx := setupMsgServer(t)
			ctx = ctx.WithBlockTime(now)
			s := current
			if tc.current != nil {
				tc.current(&s)
			}
			setSponsorship(t, f, ctx, s)
			_, err := ms.GrantSponsorship(ctx, grantMsg(func(m *types.MsgGrantSponsorship) {
				raise(m)
				if tc.mut != nil {
					tc.mut(m)
				}
			}))
			require.ErrorIs(t, err, types.ErrSponsorshipNotRaised)
		})
	}
}

func TestGrantSponsorship_Raise(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	ctx = ctx.WithBlockTime(now)
	created, exp, later, period := now.Add(-time.Hour), now.Add(time.Hour), now.Add(2*time.Hour), time.Hour
	f.corpKeeper.sponsors[sponsoredID] = testCorpID
	setSponsorship(t, f, ctx, types.Sponsorship{
		SponsorCorporationId: testCorpID, SponsoredCorporationId: sponsoredID,
		MsgTypes: []types.MsgType{mtEcosystem}, SpendLimit: uvna(100), RemainingSpend: uvna(40),
		Period: &period, Expiration: &exp, CoversFees: true, FeeSpendLimit: uvna(10), Created: created,
	})
	granter, grantee := sdk.MustAccAddressFromBech32(sponsorPolicy), sdk.MustAccAddressFromBech32(corpPolicyAddr(sponsoredID))
	require.NoError(t, f.feegrants.GrantAllowance(ctx, granter, grantee, &feegrant.BasicAllowance{SpendLimit: uvna(10)}))

	atom := sdk.NewInt64Coin("uatom", 7)
	_, err := ms.GrantSponsorship(ctx, grantMsg(func(m *types.MsgGrantSponsorship) {
		m.SpendLimit, m.Period, m.Expiration = uvna(150).Add(atom), &period, &later
		m.CoversFees, m.FeeSpendLimit = true, uvna(20)
	}))
	require.NoError(t, err)

	s := getSponsorshipOf(t, f, ctx, testCorpID)
	require.Equal(t, types.Sponsorship{
		SponsorCorporationId: testCorpID, SponsoredCorporationId: sponsoredID,
		SpendLimit: uvna(150).Add(atom), RemainingSpend: uvna(90).Add(atom),
		Period: &period, Expiration: &later, CoversFees: true, FeeSpendLimit: uvna(20), Created: created,
	}, s)
	require.Equal(t, testCorpID, f.corpKeeper.sponsors[sponsoredID])

	a, err := sponsorAllowance(t, f, ctx)
	require.NoError(t, err)
	inner, err := a.(*feegrant.AllowedMsgAllowance).GetAllowance()
	require.NoError(t, err)
	require.Equal(t, uvna(20), inner.(*feegrant.PeriodicAllowance).PeriodSpendLimit)
	_, hasNow := lastEvent(ctx, types.EventTypeGrantSponsorship)["now"]
	require.False(t, hasNow)
}

func TestGrantSponsorship_RaiseToCoverFees(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	setSponsorship(t, f, ctx, types.Sponsorship{SponsorCorporationId: testCorpID, SponsoredCorporationId: sponsoredID})

	_, err := ms.GrantSponsorship(ctx, grantMsg(func(m *types.MsgGrantSponsorship) { m.CoversFees, m.FeeSpendLimit = true, uvna(3) }))
	require.NoError(t, err)
	require.Equal(t, uvna(3), getSponsorshipOf(t, f, ctx, testCorpID).FeeSpendLimit)
	_, err = sponsorAllowance(t, f, ctx)
	require.NoError(t, err)
}

func TestGrantSponsorship_FeeLimitDroppedWithoutFees(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	_, err := ms.GrantSponsorship(ctx, grantMsg(func(m *types.MsgGrantSponsorship) { m.FeeSpendLimit = uvna(3) }))
	require.NoError(t, err)
	require.Empty(t, getSponsorshipOf(t, f, ctx, testCorpID).FeeSpendLimit)
}

func getSponsorshipOf(t *testing.T, f *fixture, ctx sdk.Context, sponsor uint64) types.Sponsorship {
	t.Helper()
	s, err := f.keeper.Sponsorships.Get(ctx, collections.Join(sponsor, sponsoredID))
	require.NoError(t, err)
	return s
}

func TestRevokeSponsorship_GroupPath(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	f.corpKeeper.sponsors[sponsoredID] = testCorpID
	setSponsorship(t, f, ctx, types.Sponsorship{SponsorCorporationId: testCorpID, SponsoredCorporationId: sponsoredID, CoversFees: true})
	granter, grantee := sdk.MustAccAddressFromBech32(sponsorPolicy), sdk.MustAccAddressFromBech32(corpPolicyAddr(sponsoredID))
	require.NoError(t, f.feegrants.GrantAllowance(ctx, granter, grantee, &feegrant.BasicAllowance{}))

	msg := &types.MsgRevokeSponsorship{Corporation: sponsorPolicy, Signer: sponsorPolicy, SponsoredCorporationId: sponsoredID}
	_, err := ms.RevokeSponsorship(ctx, msg)
	require.NoError(t, err)

	has, err := f.keeper.Sponsorships.Has(ctx, collections.Join(testCorpID, sponsoredID))
	require.NoError(t, err)
	require.False(t, has)
	_, err = sponsorAllowance(t, f, ctx)
	require.Error(t, err)
	require.Zero(t, f.corpKeeper.sponsors[sponsoredID])

	ev := lastEvent(ctx, types.EventTypeRevokeSponsorship)
	require.Equal(t, "1", ev[types.AttributeKeySponsorCorporationID])
	require.Equal(t, "2", ev[types.AttributeKeySponsoredCorporationID])
	require.Equal(t, "true", ev[types.AttributeKeySponsorCleared])
	require.Contains(t, ev["msg"], `"sponsored_corporation_id":"2"`)
}

func TestRevokeSponsorship_OtherElectionKept(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	f.corpKeeper.sponsors[sponsoredID] = sponsorID
	setSponsorship(t, f, ctx, types.Sponsorship{SponsorCorporationId: testCorpID, SponsoredCorporationId: sponsoredID})

	_, err := ms.RevokeSponsorship(ctx, &types.MsgRevokeSponsorship{Corporation: sponsorPolicy, Signer: sponsorPolicy, SponsoredCorporationId: sponsoredID})
	require.NoError(t, err)
	require.Equal(t, sponsorID, f.corpKeeper.sponsors[sponsoredID])
	require.Equal(t, "false", lastEvent(ctx, types.EventTypeRevokeSponsorship)[types.AttributeKeySponsorCleared])
}

func TestRevokeSponsorship_NotFound(t *testing.T) {
	_, ms, ctx := setupMsgServer(t)
	_, err := ms.RevokeSponsorship(ctx, &types.MsgRevokeSponsorship{Corporation: sponsorPolicy, Signer: sponsorPolicy, SponsoredCorporationId: sponsoredID})
	require.ErrorIs(t, err, types.ErrSponsorshipNotFound)
}

func TestRevokeSponsorship_Delegate(t *testing.T) {
	delegate := acc("delegate____________")
	for _, tc := range []struct {
		name     string
		msgTypes []types.MsgType
		wantErr  error
	}{
		{"authorized", []types.MsgType{mtRevokeSponsorship}, nil},
		{"not authorized for the type", []types.MsgType{mtEcosystem}, types.ErrAuthzMsgTypeNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, ms, ctx := setupMsgServer(t)
			setSponsorship(t, f, ctx, types.Sponsorship{SponsorCorporationId: testCorpID, SponsoredCorporationId: sponsoredID})
			_, err := ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
				Corporation: sponsorPolicy, Signer: sponsorPolicy, Grantee: delegate, MsgTypes: tc.msgTypes,
			})
			require.NoError(t, err)

			_, err = ms.RevokeSponsorship(ctx, &types.MsgRevokeSponsorship{Corporation: sponsorPolicy, Signer: delegate, SponsoredCorporationId: sponsoredID})
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			_, hasMsg := lastEvent(ctx, types.EventTypeRevokeSponsorship)["msg"]
			require.False(t, hasMsg)
		})
	}
}
