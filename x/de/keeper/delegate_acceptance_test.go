package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/de/types"
)

// [MOD-GF-MSG-4-3] the acceptance is stored on the authorization and read by GF-CHECK-2.
func TestSetDelegateAcceptedCGFVersion(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")
	_, err := ms.GrantDelegateAuthorization(ctx, &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee, MsgTypes: []types.MsgType{mtEcosystem},
	})
	require.NoError(t, err)

	require.NoError(t, f.keeper.SetDelegateAcceptedCGFVersion(ctx, testCorpID, grantee, 3))
	require.NoError(t, f.keeper.CheckDelegateAuthorization(ctx, testCorpID, grantee, mtEcosystem, ctx.BlockTime()))
	require.Equal(t, uint32(3), f.gf.gotAuthz.AcceptedCgfVersion)

	require.ErrorIs(t, f.keeper.SetDelegateAcceptedCGFVersion(ctx, testCorpID, acc("stranger____________"), 3), types.ErrAuthzNotFound)
	require.ErrorIs(t, f.keeper.SetDelegateAcceptedCGFVersion(ctx, testCorpID+1, grantee, 3), types.ErrAuthzNotFound)
}

// [MOD-DE-MSG-3-3] a re-grant keeps accepted_cgf_version.
func TestRegrantKeepsAcceptedCGFVersion(t *testing.T) {
	f, ms, ctx := setupMsgServer(t)
	corporation := acc("corp________________")
	grantee := acc("grantee_____________")
	grant := &types.MsgGrantDelegateAuthorization{
		Corporation: corporation, Delegate: corporation, Grantee: grantee, MsgTypes: []types.MsgType{mtEcosystem},
	}
	_, err := ms.GrantDelegateAuthorization(ctx, grant)
	require.NoError(t, err)
	require.NoError(t, f.keeper.SetDelegateAcceptedCGFVersion(ctx, testCorpID, grantee, 2))

	_, err = ms.GrantDelegateAuthorization(ctx, grant)
	require.NoError(t, err)
	require.NoError(t, f.keeper.CheckDelegateAuthorization(ctx, testCorpID, grantee, mtEcosystem, ctx.BlockTime()))
	require.Equal(t, uint32(2), f.gf.gotAuthz.AcceptedCgfVersion)
}
