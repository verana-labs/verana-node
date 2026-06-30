package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana/x/pp/types"
)

// TestMsgCreateOrUpdateParticipantSession_ValidateBasic exercises [MOD-PP-MSG-10-1].
// agent_participant_id and wallet_agent_participant_id are (*optional*): only set
// when the peer is a Verifiable User Agent, and MUST NOT be set when the peer is a
// Verifiable Service. The VS-must-not-set rule is enforced off-chain (the peer
// "MUST refuse"); on-chain the message MUST accept both 0 (VS peer) and non-zero
// (VUA peer). See verana-labs/verifiable-trust-vpr-spec spec.md MOD-PP-MSG-10-1.
func TestMsgCreateOrUpdateParticipantSession_ValidateBasic(t *testing.T) {
	validAddr := sdk.AccAddress([]byte("test_address________")).String()
	validUUID := "123e4567-e89b-12d3-a456-426614174000"

	valid := func() *types.MsgCreateOrUpdateParticipantSession {
		return &types.MsgCreateOrUpdateParticipantSession{
			Corporation:              validAddr,
			Operator:                 validAddr,
			Id:                       validUUID,
			IssuerParticipantId:      1,
			AgentParticipantId:       1,
			WalletAgentParticipantId: 1,
		}
	}

	tests := []struct {
		name    string
		mutate  func(m *types.MsgCreateOrUpdateParticipantSession)
		wantErr string
	}{
		{"valid VUA (both agent ids set)", func(m *types.MsgCreateOrUpdateParticipantSession) {}, ""},
		{"valid VS-to-VS (both agent ids zero)", func(m *types.MsgCreateOrUpdateParticipantSession) {
			m.AgentParticipantId = 0
			m.WalletAgentParticipantId = 0
		}, ""},
		{"valid agent set, wallet agent zero", func(m *types.MsgCreateOrUpdateParticipantSession) {
			m.WalletAgentParticipantId = 0
		}, ""},
		{"valid wallet agent set, agent zero", func(m *types.MsgCreateOrUpdateParticipantSession) {
			m.AgentParticipantId = 0
		}, ""},
		{"issuer and verifier both zero rejected", func(m *types.MsgCreateOrUpdateParticipantSession) {
			m.IssuerParticipantId = 0
			m.VerifierParticipantId = 0
		}, "at least one of"},
		{"invalid uuid rejected", func(m *types.MsgCreateOrUpdateParticipantSession) {
			m.Id = "garbage"
		}, "valid UUID"},
		{"invalid corporation rejected", func(m *types.MsgCreateOrUpdateParticipantSession) {
			m.Corporation = "bad"
		}, "invalid corporation"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := valid()
			tc.mutate(m)
			err := m.ValidateBasic()
			if tc.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}
