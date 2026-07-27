package keeper_test

import (
	"context"
	"testing"

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
