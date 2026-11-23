package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/verana-labs/verana-node/x/de/types"
)

func TestGenesisState_Validate(t *testing.T) {
	addr := sdk.AccAddress([]byte("operator____________")).String()
	withOA := func(mts ...types.MsgType) *types.GenesisState {
		return &types.GenesisState{Params: types.DefaultParams(), OperatorAuthorizations: []types.OperatorAuthorization{
			{Id: 1, CorporationId: 1, Operator: addr, MsgTypes: mts},
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
		{desc: "operator authorization delegable value", genState: withOA(types.MsgType_MSG_TYPE_STORE_DIGEST), valid: true},
		{desc: "operator authorization issuance value", genState: withOA(types.MsgType_MSG_TYPE_RECORD_CREDENTIAL_ISSUANCE), valid: false},
		{desc: "operator authorization without values", genState: withOA(), valid: false},
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
