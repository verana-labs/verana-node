package keeper_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/x/co/keeper"
	"github.com/verana-labs/verana-node/x/co/types"
)

// fakeDIDOwner reports a given owner only for a specific did.
type fakeDIDOwner struct {
	did   string
	owner uint64
}

func (f fakeDIDOwner) ResolveDIDOwner(_ context.Context, did string) (uint64, bool, error) {
	if did == f.did {
		return f.owner, true, nil
	}
	return 0, false, nil
}

func TestUpdateCorporation_RejectsDIDOwnedByForeignEcosystem(t *testing.T) {
	grp := &mockGroup{policy: tkPolicy}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, &mockGF{})
	k.SetEcosystemDIDResolver(fakeDIDOwner{did: "did:example:foreign", owner: 99})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateCorporation(ctx, validCreateMsg(t)) // corp 1, did:example:1
	require.NoError(t, err)

	_, err = ms.UpdateCorporation(ctx, &types.MsgUpdateCorporation{
		Corporation: tkPolicy,
		Operator:    tkOp,
		Did:         "did:example:foreign",
	})
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)
}

// Rotating to a did already bound by an entity of the SAME corp is allowed.
func TestUpdateCorporation_AllowsDIDOwnedBySelf(t *testing.T) {
	grp := &mockGroup{policy: tkPolicy}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, &mockGF{})
	k.SetParticipantDIDResolver(fakeDIDOwner{did: "did:example:self", owner: 1})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateCorporation(ctx, validCreateMsg(t)) // corp 1
	require.NoError(t, err)

	_, err = ms.UpdateCorporation(ctx, &types.MsgUpdateCorporation{
		Corporation: tkPolicy,
		Operator:    tkOp,
		Did:         "did:example:self",
	})
	require.NoError(t, err)
}

// Creating a corporation whose did is already claimed by any ec/pp entry aborts.
func TestCreateCorporation_RejectsDIDOwnedElsewhere(t *testing.T) {
	grp := &mockGroup{policy: tkPolicy}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, &mockGF{})
	msg := validCreateMsg(t)
	k.SetEcosystemDIDResolver(fakeDIDOwner{did: msg.Did, owner: 5})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateCorporation(ctx, msg)
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)
}

// The participant leg of the create-time cross-check.
func TestCreateCorporation_RejectsDIDOwnedByForeignParticipant(t *testing.T) {
	grp := &mockGroup{policy: tkPolicy}
	k, ctx := keepertest.CoKeeper(t, &mockDelegation{}, grp, &mockGF{})
	msg := validCreateMsg(t)
	k.SetParticipantDIDResolver(fakeDIDOwner{did: msg.Did, owner: 5})
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.CreateCorporation(ctx, msg)
	require.ErrorIs(t, err, types.ErrDIDOwnershipConflict)
}
