package keeper_test

import (
	"context"
	"testing"
	"time"

	"github.com/verana-labs/verana-node/x/ec/keeper"
	"github.com/verana-labs/verana-node/x/ec/types"

	"github.com/stretchr/testify/require"
)

// fakeDIDResolver reports a given owner only for a specific did.
type fakeDIDResolver struct {
	did   string
	owner uint64
}

func (f fakeDIDResolver) ResolveDIDOwner(_ context.Context, did string) (uint64, bool, error) {
	if did == f.did {
		return f.owner, true, nil
	}
	return 0, false, nil
}

func TestCreateEcosystem_RejectsDIDOwnedByForeignParticipant(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	k.SetParticipantDIDResolver(fakeDIDResolver{did: "did:example:ec1", owner: 99})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)
}

func TestCreateEcosystem_AllowsDIDOwnedBySelfParticipant(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	k.SetParticipantDIDResolver(fakeDIDResolver{did: "did:example:ec1", owner: 1}) // same corp
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t))
	require.NoError(t, err)
}

// UpdateEcosystem (MOD-ES-MSG-2-2-1) rejects rotating to a did a foreign
// corporation's participant controls.
func TestUpdateEcosystem_RejectsDIDOwnedByForeignParticipant(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	k.SetParticipantDIDResolver(fakeDIDResolver{did: "did:example:rotated", owner: 99})
	ms := keeper.NewMsgServerImpl(k)
	ctx = ctx.WithBlockTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t)) // ec 1, did:example:ec1
	require.NoError(t, err)

	_, err = ms.UpdateEcosystem(ctx, &types.MsgUpdateEcosystem{
		Corporation: tkCorp, Operator: tkOp, Id: 1, Did: "did:example:rotated",
	})
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)
}

// The corporation leg of assertNoForeignDIDOwnerES (fails if that branch is removed).
func TestCreateEcosystem_RejectsDIDOwnedByForeignCorporation(t *testing.T) {
	co := newMockCorporation()
	co.register(tkCorp, 1)
	co.registerDID("did:example:ec1", 2) // corp 2 owns this as its Corporation.did
	k, ctx := ecKeeper(t, &mockDelegation{}, co, &mockGF{})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateEcosystem(ctx, validCreateMsg(t)) // did:example:ec1, signer corp 1
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)
}
