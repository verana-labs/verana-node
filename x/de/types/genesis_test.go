package types_test

import (
	"time"

	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana-node/x/de/types"
)

func TestGenesisState_Validate(t *testing.T) {
	addr := sdk.AccAddress([]byte("delegate____________")).String()
	withOA := func(mts ...types.MsgType) *types.GenesisState {
		return &types.GenesisState{Params: types.DefaultParams(), DelegateAuthorizations: []types.DelegateAuthorization{
			{Id: 1, CorporationId: 1, Delegate: addr, MsgTypes: mts},
		}}
	}
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
