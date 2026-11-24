package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	"github.com/verana-labs/verana-node/x/gf/keeper"
	"github.com/verana-labs/verana-node/x/gf/types"
)

func TestQueryGetGovernanceFrameworkVersion_NilRequest(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.GetGovernanceFrameworkVersion(ctx, nil)
	require.Error(t, err)
}

func TestQueryGetGovernanceFrameworkVersion_NotFound(t *testing.T) {
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 999})
	require.Error(t, err)
}

func TestQueryGetGovernanceFrameworkVersion_PreferredLanguageFallbackWhenNoMatch(t *testing.T) {
	corp := &mockCorporation{
		view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en"},
		found: true,
	}
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, corp)
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testDelegate, 0, 1))
	require.NoError(t, err)
	frMsg := validMsg(testCorp, testDelegate, 0, 1)
	frMsg.DocLanguage = "fr"
	frMsg.DocUrl = "https://example.com/gf-v1-fr.html"
	_, err = ms.AddGovernanceFrameworkDocument(ctx, frMsg)
	require.NoError(t, err)

	qs := keeper.NewQueryServerImpl(k)
	// "es" has no document: the one in the corporation's language is returned.
	resp, err := qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{
		Id:                1,
		PreferredLanguage: "es",
	})
	require.NoError(t, err)
	require.Len(t, resp.Version.Documents, 1, "preferred_language set must yield one document per version")
	require.Equal(t, "en", resp.Version.Documents[0].Language)

	// With no document in the corporation's language either, the earliest one.
	corp.view.Language = "de"
	resp, err = qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 1, PreferredLanguage: "es"})
	require.NoError(t, err)
	require.Len(t, resp.Version.Documents, 1)
	require.Equal(t, uint64(1), resp.Version.Documents[0].Id)
}

// [MOD-GF-QRY-1-3] an ecosystem version falls back to the ecosystem's language.
func TestQueryGetGovernanceFrameworkVersion_EcosystemLanguageFallback(t *testing.T) {
	corp := &mockCorporation{view: types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en"}, found: true}
	eco := &mockEcosystem{view: types.EcosystemView{Id: 7, CorporationID: 1, Language: "fr"}, found: true}
	k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco, corp)
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testDelegate, 7, 1))
	require.NoError(t, err)
	frMsg := validMsg(testCorp, testDelegate, 7, 1)
	frMsg.DocLanguage = "fr"
	_, err = ms.AddGovernanceFrameworkDocument(ctx, frMsg)
	require.NoError(t, err)

	eco.gotID = 0
	resp, err := keeper.NewQueryServerImpl(k).GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 1, PreferredLanguage: "es"})
	require.NoError(t, err)
	require.Equal(t, uint64(7), eco.gotID)
	require.Len(t, resp.Version.Documents, 1)
	require.Equal(t, "fr", resp.Version.Documents[0].Language)
}

// [MOD-GF-QRY-1-3] a network version has no owner language: preferred, else the earliest.
func TestQueryGetGovernanceFrameworkVersion_NetworkFallback(t *testing.T) {
	k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	addNetworkVersion(t, k, ctx, 2, 0)
	ms := keeper.NewMsgServerImpl(k)
	_, err := ms.AddGovernanceFrameworkDocument(ctx.WithBlockTime(genesisTime.Add(time.Hour)), networkDoc(k.GetAuthority(), "de", 2))
	require.NoError(t, err)

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 2, PreferredLanguage: "de"})
	require.NoError(t, err)
	require.Equal(t, "de", resp.Version.Documents[0].Language)
	resp, err = qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 2, PreferredLanguage: "es"})
	require.NoError(t, err)
	require.Len(t, resp.Version.Documents, 1)
	require.Equal(t, "en", resp.Version.Documents[0].Language)
	require.Zero(t, resp.Version.EcosystemId)
	require.Zero(t, resp.Version.CorporationId)
}

func TestQueryGetGovernanceFrameworkVersion(t *testing.T) {
	t.Run("MOD-GF-QRY-1: returns GFV with docs, preferred language filter applied", func(t *testing.T) {
		corp := &mockCorporation{
			view:  types.CorporationView{Id: 1, PolicyAddress: testCorp, Language: "en", ActiveVersion: 0},
			found: true,
		}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, corp)
		ms := keeper.NewMsgServerImpl(k)
		// Setup: add 2 docs (en, fr) under v1.
		_, err := ms.AddGovernanceFrameworkDocument(ctx, validMsg(testCorp, testDelegate, 0, 1))
		require.NoError(t, err)
		frMsg := validMsg(testCorp, testDelegate, 0, 1)
		frMsg.DocLanguage = "fr"
		frMsg.DocUrl = "https://example.com/gf-v1-fr.html"
		_, err = ms.AddGovernanceFrameworkDocument(ctx, frMsg)
		require.NoError(t, err)

		qs := keeper.NewQueryServerImpl(k)
		// No filter — both docs.
		resp, err := qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 1})
		require.NoError(t, err)
		require.Len(t, resp.Version.Documents, 2)

		// Preferred language "fr" — only the fr doc.
		respFR, err := qs.GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 1, PreferredLanguage: "fr"})
		require.NoError(t, err)
		require.Len(t, respFR.Version.Documents, 1)
		require.Equal(t, "fr", respFR.Version.Documents[0].Language)
	})
}
