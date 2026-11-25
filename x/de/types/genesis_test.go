package types_test

import (
	"strings"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana-node/x/de/types"
)

func TestGenesisState_Validate(t *testing.T) {
	addr := sdk.AccAddress([]byte("delegate____________")).String()
	withOA := func(mts ...types.MsgType) *types.GenesisState {
		return &types.GenesisState{Params: types.DefaultParams(), DelegateAuthorizations: []types.DelegateAuthorization{
			{CorporationId: 1, Delegate: addr, MsgTypes: mts},
		}}
	}
	withDAs := func(das ...types.DelegateAuthorization) *types.GenesisState {
		for i := range das {
			das[i].MsgTypes = []types.MsgType{types.MsgType_MSG_TYPE_STORE_DIGEST}
		}
		return &types.GenesisState{Params: types.DefaultParams(), DelegateAuthorizations: das}
	}
	limit := sdk.NewCoins(sdk.NewInt64Coin("uvna", 10))
	withFG := func(mts ...types.MsgType) *types.GenesisState {
		return &types.GenesisState{Params: types.DefaultParams(), FeeGrants: []types.FeeGrant{
			{GrantorCorporationId: 1, Grantee: addr, MsgTypes: mts},
		}}
	}
	withRecord := func(mts ...types.MsgType) *types.GenesisState {
		return &types.GenesisState{Params: types.DefaultParams(), VsOperatorAuthorizations: []types.VSOperatorAuthorization{
			{Id: 1, CorporationId: 1, VsOperator: addr, Records: []types.ParticipantAuthorizationRecord{{ParticipantId: 1, MsgTypes: mts}}},
		}}
	}
	period, exp := time.Hour, time.Unix(100, 0)
	withSp := func(s ...types.Sponsorship) *types.GenesisState {
		return &types.GenesisState{Params: types.DefaultParams(), Sponsorships: s}
	}
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc:     "valid genesis state",
			genState: &types.GenesisState{Params: types.DefaultParams()},
			valid:    true,
		},
		{
			desc:     "zero fee period is invalid",
			genState: &types.GenesisState{},
			valid:    false,
		},
		{
			desc: "queue entry without participant is invalid",
			genState: &types.GenesisState{
				Params:         types.DefaultParams(),
				WindowEndQueue: []types.WindowEndQueueEntry{{ParticipantId: 0}},
			},
			valid: false,
		},
		{desc: "delegate authorization delegable value", genState: withOA(types.MsgType_MSG_TYPE_STORE_DIGEST), valid: true},
		{desc: "delegate authorization issuance value", genState: withOA(types.MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE), valid: false},
		{desc: "delegate authorization without values", genState: withOA(), valid: false},
		{desc: "delegate authorization without corporation", genState: withDAs(types.DelegateAuthorization{Delegate: addr}), valid: false},
		{desc: "delegate authorization invalid delegate", genState: withDAs(types.DelegateAuthorization{CorporationId: 1, Delegate: "x"}), valid: false},
		{desc: "delegate authorizations of two corporations", genState: withDAs(
			types.DelegateAuthorization{CorporationId: 1, Delegate: addr},
			types.DelegateAuthorization{CorporationId: 2, Delegate: addr},
		), valid: true},
		{desc: "delegate authorization duplicate key", genState: withDAs(
			types.DelegateAuthorization{CorporationId: 1, Delegate: addr},
			types.DelegateAuthorization{CorporationId: 1, Delegate: strings.ToUpper(addr)},
		), valid: false},
		{desc: "delegate authorization cycle", genState: withDAs(types.DelegateAuthorization{
			CorporationId: 1, Delegate: addr, SpendLimit: limit, RemainingSpend: limit, Expiration: &exp, Period: &period,
		}), valid: true},
		{desc: "delegate authorization period without expiration", genState: withDAs(types.DelegateAuthorization{
			CorporationId: 1, Delegate: addr, SpendLimit: limit, RemainingSpend: limit, Period: &period,
		}), valid: false},
		{desc: "delegate authorization remaining without limit", genState: withDAs(types.DelegateAuthorization{
			CorporationId: 1, Delegate: addr, RemainingSpend: limit,
		}), valid: false},
		{desc: "delegate authorization spent", genState: withDAs(types.DelegateAuthorization{
			CorporationId: 1, Delegate: addr, SpendLimit: limit,
		}), valid: true},
		{desc: "delegate authorization remaining above limit", genState: withDAs(types.DelegateAuthorization{
			CorporationId: 1, Delegate: addr, SpendLimit: limit, RemainingSpend: limit.Add(limit...),
		}), valid: false},
		{desc: "delegate authorization unsorted limit", genState: withDAs(types.DelegateAuthorization{
			CorporationId: 1, Delegate: addr, SpendLimit: sdk.Coins{sdk.NewInt64Coin("uvna", 1), sdk.NewInt64Coin("uatom", 1)},
		}), valid: false},
		{desc: "delegate authorization period without limit", genState: withDAs(types.DelegateAuthorization{
			CorporationId: 1, Delegate: addr, Expiration: &exp, Period: &period,
		}), valid: false},
		{desc: "delegate authorization zero period", genState: withDAs(types.DelegateAuthorization{
			CorporationId: 1, Delegate: addr, SpendLimit: limit, RemainingSpend: limit, Expiration: &exp, Period: new(time.Duration),
		}), valid: false},
		{desc: "fee grant session value", genState: withFG(types.MsgType_MSG_TYPE_RECORD_CREDENTIAL_PRESENTATION), valid: true},
		{desc: "fee grant unspecified value", genState: withFG(types.MsgType_MSG_TYPE_UNSPECIFIED), valid: false},
		{desc: "record vs value", genState: withRecord(types.MsgType_MSG_TYPE_VALIDATE_ONBOARDING), valid: true},
		{desc: "record non-vs value", genState: withRecord(types.MsgType_MSG_TYPE_CREATE_ECOSYSTEM), valid: false},
		{desc: "sponsorship", genState: withSp(types.Sponsorship{SponsorCorporationId: 1, SponsoredCorporationId: 2}), valid: true},
		{desc: "sponsorship of itself", genState: withSp(types.Sponsorship{SponsorCorporationId: 1, SponsoredCorporationId: 1}), valid: false},
		{desc: "sponsorship without sponsor", genState: withSp(types.Sponsorship{SponsoredCorporationId: 1}), valid: false},
		{desc: "duplicate sponsorship", genState: withSp(
			types.Sponsorship{SponsorCorporationId: 1, SponsoredCorporationId: 2},
			types.Sponsorship{SponsorCorporationId: 1, SponsoredCorporationId: 2},
		), valid: false},
		{desc: "sponsorship value outside its list", genState: withSp(types.Sponsorship{
			SponsorCorporationId: 1, SponsoredCorporationId: 2, MsgTypes: []types.MsgType{types.MsgType_MSG_TYPE_STORE_DIGEST},
		}), valid: false},
		{desc: "cycling sponsorship without expiration", genState: withSp(types.Sponsorship{
			SponsorCorporationId: 1, SponsoredCorporationId: 2, Period: &period,
		}), valid: false},
		{desc: "cycling sponsorship", genState: withSp(types.Sponsorship{
			SponsorCorporationId: 1, SponsoredCorporationId: 2, Period: &period, Expiration: &exp,
		}), valid: true},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
