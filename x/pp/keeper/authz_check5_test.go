package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	cotypes "github.com/verana-labs/verana-node/x/co/types"
	"github.com/verana-labs/verana-node/x/pp/keeper"
	"github.com/verana-labs/verana-node/x/pp/types"
)

// AUTHZ-CHECK-5 runs first on every MOD-PP message: an unregistered
// corporation aborts before any delegate or VS delegate check, and the
// delegate check receives the resolved corporation id.
func TestAuthzCheck5_Participant(t *testing.T) {
	corporation := sdk.AccAddress([]byte("pp_corporation______")).String()
	delegate := sdk.AccAddress([]byte("pp_delegate_________")).String()
	signer := delegate
	type run func(ms types.MsgServer, ctx sdk.Context) error
	cases := []struct {
		name string
		run  run
		// delegateFirst: AUTHZ-CHECK-1 runs before any method-specific check.
		delegateFirst bool
	}{
		{"start", func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.StartParticipantOP(ctx, &types.MsgStartParticipantOP{Corporation: corporation, Signer: signer, ValidatorParticipantId: 1})
			return err
		}, true},
		{"renew", func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.RenewParticipantOP(ctx, &types.MsgRenewParticipantOP{Corporation: corporation, Signer: signer, Id: 1})
			return err
		}, true},
		{"validate", func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.SetParticipantOPToValidated(ctx, &types.MsgSetParticipantOPToValidated{Corporation: corporation, Signer: signer, Id: 1})
			return err
		}, false},
		{"cancel", func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.CancelParticipantOPLastRequest(ctx, &types.MsgCancelParticipantOPLastRequest{Corporation: corporation, Signer: signer, Id: 1})
			return err
		}, true},
		{"create root", func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.CreateRootParticipant(ctx, &types.MsgCreateRootParticipant{Corporation: corporation, Signer: signer, SchemaId: 1})
			return err
		}, true},
		{"set effective until", func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.SetParticipantEffectiveUntil(ctx, &types.MsgSetParticipantEffectiveUntil{Corporation: corporation, Signer: signer, Id: 1})
			return err
		}, true},
		{"revoke", func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.RevokeParticipant(ctx, &types.MsgRevokeParticipant{Corporation: corporation, Signer: signer, Id: 1})
			return err
		}, true},
		{"session", func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.CreateOrUpdateParticipantSession(ctx, &types.MsgCreateOrUpdateParticipantSession{Corporation: corporation, Signer: signer, IssuerParticipantId: 1})
			return err
		}, false},
		{"slash", func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.SlashParticipantTrustDeposit(ctx, &types.MsgSlashParticipantTrustDeposit{Corporation: corporation, Signer: signer, Id: 1, Amount: 1})
			return err
		}, true},
		{"repay", func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.RepayParticipantSlashedTrustDeposit(ctx, &types.MsgRepayParticipantSlashedTrustDeposit{Corporation: corporation, Signer: signer, Id: 1})
			return err
		}, true},
		{"self create", func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{Corporation: corporation, Signer: signer, ValidatorParticipantId: 1})
			return err
		}, true},
		{"trigger resolver", func(ms types.MsgServer, ctx sdk.Context) error {
			_, err := ms.TriggerResolver(ctx, &types.MsgTriggerResolver{Corporation: corporation, Signer: signer, Id: 1})
			return err
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name+": unregistered corporation aborts first", func(t *testing.T) {
			k, _, _, co, ctx, del := keepertest.ParticipantKeeper(t)
			co.Unregistered[corporation] = true
			require.ErrorIs(t, tc.run(keeper.NewMsgServerImpl(k), ctx), cotypes.ErrCorporationNotRegistered)
			require.Empty(t, del.DelegateAuthCorpIDs)
			require.Empty(t, del.VSOACheckMsgTypes)
		})
		if !tc.delegateFirst {
			continue
		}
		t.Run(tc.name+": AUTHZ-CHECK-1 gets the resolved id", func(t *testing.T) {
			k, _, ec, _, ctx, del := keepertest.ParticipantKeeper(t)
			corpID := ec.RegisterCorp(corporation)
			_ = tc.run(keeper.NewMsgServerImpl(k), ctx)
			require.Equal(t, []uint64{corpID}, del.DelegateAuthCorpIDs)
		})
		t.Run(tc.name+": group-proposal path skips AUTHZ-CHECK-1", func(t *testing.T) {
			k, _, ec, _, ctx, del := keepertest.ParticipantKeeper(t)
			ec.RegisterCorp(corporation)
			signer = corporation
			defer func() { signer = delegate }()
			_ = tc.run(keeper.NewMsgServerImpl(k), ctx)
			require.Empty(t, del.DelegateAuthCorpIDs)
		})
	}
}
