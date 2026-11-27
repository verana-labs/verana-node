package app

import (
	"testing"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/stretchr/testify/require"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
	dekeeper "github.com/verana-labs/verana-node/x/de/keeper"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	tdtypes "github.com/verana-labs/verana-node/x/td/types"
)

// The sponsored-fees rule through the real ante chain: exec signs with the
// corporation as fee granter; its sponsor pays when its balance and its
// allowance to the corporation allow, else the corporation pays.
func TestSponsoredFees(t *testing.T) {
	sponsor := sdk.AccAddress([]byte("sponsor_policy______"))
	cases := []struct {
		name       string
		coversFees bool
		sponsorBal int64
		allowance  feegrant.FeeAllowanceI
		payer      string
	}{
		{"sponsor pays", true, 5000, &feegrant.BasicAllowance{SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uvna", 1500))}, "sponsor"},
		{"sponsor allowance short", true, 5000, &feegrant.BasicAllowance{SpendLimit: sdk.NewCoins(sdk.NewInt64Coin("uvna", 999))}, "corp"},
		{"sponsor balance short", true, 999, &feegrant.BasicAllowance{}, "corp"},
		{"sponsorship pays no fees", false, 5000, &feegrant.BasicAllowance{}, "corp"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := newFeeGranterEnv(t)
			a := e.app
			ctx := e.nextBlockCtx()
			require.NoError(t, a.BankKeeper.SendCoins(ctx, e.other, sponsor, sdk.NewCoins(sdk.NewInt64Coin("uvna", tc.sponsorBal))))
			sp := cotypes.Corporation{Id: 2, PolicyAddress: sponsor.String(), Did: "did:example:sponsor"}
			require.NoError(t, a.CoKeeper.Corporation.Set(ctx, sp.Id, sp))
			require.NoError(t, a.CoKeeper.CorporationByPolicyAddr.Set(ctx, sp.PolicyAddress, sp.Id))
			corp, err := a.CoKeeper.Corporation.Get(ctx, 1)
			require.NoError(t, err)
			corp.SponsorCorporationId = sp.Id
			require.NoError(t, a.CoKeeper.Corporation.Set(ctx, corp.Id, corp))
			require.NoError(t, a.DeKeeper.Sponsorships.Set(ctx, collections.Join(sp.Id, corp.Id), detypes.Sponsorship{
				SponsorCorporationId: sp.Id, SponsoredCorporationId: corp.Id, CoversFees: tc.coversFees,
			}))
			require.NoError(t, a.FeeGrantKeeper.GrantAllowance(ctx, sponsor, e.corp, tc.allowance))
			e.finalize(t)

			payer, other := e.corp, sponsor
			if tc.payer == "sponsor" {
				payer, other = sponsor, e.corp
			}
			send := &banktypes.MsgSend{FromAddress: e.execAddr().String(), ToAddress: e.other.String(), Amount: sdk.NewCoins(sdk.NewInt64Coin("uvna", 1))}
			bz := e.signedTx(t, e.exec, e.corp, send)
			payerBefore, otherBefore := e.uvna(payer), e.uvna(other)
			allowanceBefore, err := a.FeeGrantKeeper.GetAllowance(e.app.NewContext(true), sponsor, e.corp)
			require.NoError(t, err)

			res := e.finalize(t, bz)[0]
			require.Zero(t, res.Code, res.Log)
			require.Equal(t, payerBefore-1000, e.uvna(payer))
			require.Equal(t, otherBefore, e.uvna(other))
			allowanceAfter, err := a.FeeGrantKeeper.GetAllowance(e.app.NewContext(true), sponsor, e.corp)
			if tc.payer == "sponsor" {
				require.NoError(t, err)
				require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("uvna", 500)), allowanceAfter.(*feegrant.BasicAllowance).SpendLimit)
				return
			}
			require.NoError(t, err)
			require.Equal(t, allowanceBefore, allowanceAfter)
		})
	}
}

// A covers_fees sponsorship allowance covers the message types of max_gas_per_msg_type.
func TestGrantSponsorship_AllowanceFilter(t *testing.T) {
	e := newFeeGranterEnv(t)
	a := e.app
	ctx := e.nextBlockCtx()
	sponsor := sdk.AccAddress([]byte("sponsor_policy______"))
	sp := cotypes.Corporation{Id: 9, PolicyAddress: sponsor.String(), Did: "did:example:sponsor"}
	require.NoError(t, a.CoKeeper.Corporation.Set(ctx, sp.Id, sp))
	require.NoError(t, a.CoKeeper.CorporationByPolicyAddr.Set(ctx, sp.PolicyAddress, sp.Id))
	params := a.TrustdepositKeeper.GetParams(ctx)
	params.MaxGasPerMsgType = []tdtypes.MaxGasEntry{
		{MsgTypeUrl: "/verana.ec.v1.MsgCreateEcosystem", MaxGas: 300000},
		{MsgTypeUrl: "/cosmos.group.v1.MsgVote", MaxGas: 100000},
	}
	require.NoError(t, a.TrustdepositKeeper.SetParams(ctx, params))

	_, err := dekeeper.NewMsgServerImpl(a.DeKeeper).GrantSponsorship(ctx, &detypes.MsgGrantSponsorship{
		Signer: sponsor.String(), SponsoredCorporationId: 4, CoversFees: true,
	})
	require.NoError(t, err)

	got, err := a.FeeGrantKeeper.GetAllowance(ctx, sponsor, e.corp)
	require.NoError(t, err)
	allowed, ok := got.(*feegrant.AllowedMsgAllowance)
	require.True(t, ok)
	require.Equal(t, []string{"/verana.ec.v1.MsgCreateEcosystem", "/cosmos.group.v1.MsgVote"}, allowed.AllowedMessages)
}
