package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana-node/x/de/types"
)

func TestGenesisState_Validate(t *testing.T) {
	addr := sdk.AccAddress([]byte("delegate____________")).String()
	withOA := func(mts ...types.MsgType) *types.GenesisState {
		return &types.GenesisState{DelegateAuthorizations: []types.DelegateAuthorization{
			{Id: 1, CorporationId: 1, Delegate: addr, MsgTypes: mts},
		}}
	}
	withFG := func(mts ...types.MsgType) *types.GenesisState {
		return &types.GenesisState{FeeGrants: []types.FeeGrant{
			{GrantorCorporationId: 1, Grantee: addr, MsgTypes: mts},
		}}
	}
	withRecord := func(mts ...types.MsgType) *types.GenesisState {
		return &types.GenesisState{VsOperatorAuthorizations: []types.VSOperatorAuthorization{
			{Id: 1, CorporationId: 1, VsOperator: addr, Records: []types.ParticipantAuthorizationRecord{{ParticipantId: 1, MsgTypes: mts}}},
		}}
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
			desc:     "empty genesis state",
			genState: &types.GenesisState{},
			valid:    true,
		},
		{
			desc: "queue entry without participant is invalid",
			genState: &types.GenesisState{
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
