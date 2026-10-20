package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/collections"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	cstypes "github.com/verana-labs/verana-node/x/cs/types"
	"github.com/verana-labs/verana-node/x/pp/keeper"
	"github.com/verana-labs/verana-node/x/pp/types"
)

func TestIndexParticipant_RootBySchemaCorp(t *testing.T) {
	k, _, _, _, ctx, _ := keepertest.ParticipantKeeper(t)

	rootID, err := k.CreateParticipant(ctx, types.Participant{SchemaId: 3, CorporationId: 7, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:root"})
	require.NoError(t, err)
	leafID, err := k.CreateParticipant(ctx, types.Participant{SchemaId: 3, CorporationId: 7, Role: types.ParticipantRole_ISSUER, Did: "did:example:leaf"})
	require.NoError(t, err)

	has, err := k.RootParticipantBySchemaCorp.Has(ctx, collections.Join3(uint64(3), uint64(7), rootID))
	require.NoError(t, err)
	require.True(t, has)
	has, err = k.RootParticipantBySchemaCorp.Has(ctx, collections.Join3(uint64(3), uint64(7), leafID))
	require.NoError(t, err)
	require.False(t, has)
}

func TestIndexParticipant_UnrepaidSlashFollowsDeposits(t *testing.T) {
	k, _, _, _, ctx, _ := keepertest.ParticipantKeeper(t)
	now := ctx.BlockTime()

	id, err := k.CreateParticipant(ctx, types.Participant{SchemaId: 3, CorporationId: 7, Did: "did:example:a", Slashed: &now, SlashedDeposit: 100})
	require.NoError(t, err)
	key := collections.Join3(uint64(7), uint64(3), id)

	has, err := k.UnrepaidSlashByCorp.Has(ctx, key)
	require.NoError(t, err)
	require.True(t, has)

	p, err := k.GetParticipantByID(ctx, id)
	require.NoError(t, err)
	p.RepaidDeposit = 100
	require.NoError(t, k.UpdateParticipant(ctx, p))
	has, err = k.UnrepaidSlashByCorp.Has(ctx, key)
	require.NoError(t, err)
	require.False(t, has)

	p.SlashedDeposit = 150
	require.NoError(t, k.UpdateParticipant(ctx, p))
	has, err = k.UnrepaidSlashByCorp.Has(ctx, key)
	require.NoError(t, err)
	require.True(t, has)
}

func TestIndexParticipant_UnslashedHasNoRow(t *testing.T) {
	k, _, _, _, ctx, _ := keepertest.ParticipantKeeper(t)

	id, err := k.CreateParticipant(ctx, types.Participant{SchemaId: 3, CorporationId: 7, Did: "did:example:a"})
	require.NoError(t, err)
	p, err := k.GetParticipantByID(ctx, id)
	require.NoError(t, err)
	require.NoError(t, k.UpdateParticipant(ctx, p))

	has, err := k.UnrepaidSlashByCorp.Has(ctx, collections.Join3(uint64(7), uint64(3), id))
	require.NoError(t, err)
	require.False(t, has)
}

// Self-create runs the overlap and unrepaid-slash checks; its gas must not grow
// with the number of unrelated Participants.
func TestSelfCreateParticipant_GasIndependentOfStoreSize(t *testing.T) {
	gasFor := func(unrelated int) uint64 {
		k, csKeeper, trkKeeper, _, ctx, _, _, _ := keepertest.ParticipantKeeperWithTrustDeposit(t)
		ms := keeper.NewMsgServerImpl(k)
		now := time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC)
		ctx = ctx.WithBlockTime(now)
		past := now.Add(-time.Hour)
		future := now.Add(time.Hour)

		validatorAddr := sdk.AccAddress([]byte("gas_validator_addr__")).String()
		eco := trkKeeper.CreateMockEcosystem(validatorAddr, "did:example:gas-eco")
		csKeeper.UpdateMockCredentialSchema(1, eco,
			cstypes.IssuerOnboardingMode_ISSUER_ONBOARDING_MODE_OPEN,
			cstypes.VerifierOnboardingMode_VERIFIER_ONBOARDING_MODE_OPEN)
		rootID, err := k.CreateParticipant(ctx, types.Participant{
			SchemaId: 1, Role: types.ParticipantRole_ECOSYSTEM, Did: "did:example:gas-root",
			CorporationId: trkKeeper.RegisterCorp(validatorAddr), Created: &now, Modified: &now,
			OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
		})
		require.NoError(t, err)

		applicant := sdk.AccAddress([]byte("gas_applicant_addr__")).String()
		applicantCorp := trkKeeper.RegisterCorp(applicant)
		for i := 0; i < unrelated; i++ {
			_, err := k.CreateParticipant(ctx, types.Participant{
				SchemaId: 1, Role: types.ParticipantRole_HOLDER, Did: fmt.Sprintf("did:example:gas-%d", i),
				CorporationId: applicantCorp, Created: &now, Modified: &now,
				ValidatorParticipantId: rootID, OpState: types.OnboardingState_VALIDATED, EffectiveFrom: &past,
			})
			require.NoError(t, err)
		}

		ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
		_, err = ms.SelfCreateParticipant(ctx, &types.MsgSelfCreateParticipant{
			Corporation: applicant, Operator: applicant,
			Role: types.ParticipantRole_ISSUER, ValidatorParticipantId: rootID,
			Did: "did:example:gas-applicant", EffectiveFrom: &future,
		})
		require.NoError(t, err)
		return ctx.GasMeter().GasConsumed()
	}

	small := gasFor(10)
	large := gasFor(2000)
	require.Less(t, large, small*2, "gas grew with store size: %d -> %d", small, large)
}
