package keeper_test

import (
	"testing"
	"time"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"

	"github.com/verana-labs/verana-node/x/gf/keeper"
	"github.com/verana-labs/verana-node/x/gf/types"
)

func networkDoc(signer, lang string, version uint32) *types.MsgAddGovernanceFrameworkDocument {
	m := validMsg("", signer, 0, version)
	m.Network = true
	m.DocLanguage = lang
	return m
}

// [MOD-GF-MSG-1] network path.
func TestAddGovernanceFrameworkDocumentNetwork(t *testing.T) {
	t.Run("only the council account signs it", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		_, err := keeper.NewMsgServerImpl(k).AddGovernanceFrameworkDocument(ctx, networkDoc(testDelegate, "en", 2))
		require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
	})

	t.Run("a new version follows the version in force", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		ms := keeper.NewMsgServerImpl(k)
		for _, v := range []uint32{1, 3} {
			_, err := ms.AddGovernanceFrameworkDocument(ctx, networkDoc(k.GetAuthority(), "en", v))
			require.ErrorIs(t, err, types.ErrInvalidVersion)
		}

		_, err := ms.AddGovernanceFrameworkDocument(ctx, networkDoc(k.GetAuthority(), "en", 2))
		require.NoError(t, err)
		ng, err := k.NetworkGovernance.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, []uint64{1, 2}, ng.NetworkGovernanceFrameworkVersions)
		gfv, err := k.GFVersion.Get(ctx, 2)
		require.NoError(t, err)
		require.Equal(t, types.GovernanceFrameworkVersion{Id: 2, Created: genesisTime, Version: 2}, gfv)

		attrs := eventAttrs(ctx, types.EventTypeAddGovernanceFrameworkDocument)
		require.Equal(t, "2", attrs[types.AttributeKeyGFVersionID])
		require.Equal(t, "2", attrs[types.AttributeKeyGFDocID])
		require.Contains(t, attrs["msg"], `"signer":"`+k.GetAuthority()+`"`)
	})

	t.Run("documents go to the draft until it is active", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		ms := keeper.NewMsgServerImpl(k)
		addNetworkVersion(t, k, ctx, 2, 0)

		_, err := ms.AddGovernanceFrameworkDocument(ctx, networkDoc(k.GetAuthority(), "fr", 2))
		require.NoError(t, err)
		_, err = ms.AddGovernanceFrameworkDocument(ctx, networkDoc(k.GetAuthority(), "en", 3))
		require.ErrorIs(t, err, types.ErrInvalidVersion)

		ng, err := k.NetworkGovernance.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, []uint64{1, 2}, ng.NetworkGovernanceFrameworkVersions)
		resp, err := keeper.NewQueryServerImpl(k).GetGovernanceFrameworkVersion(ctx, &types.QueryGetGovernanceFrameworkVersionRequest{Id: 2})
		require.NoError(t, err)
		require.Len(t, resp.Version.Documents, 2)
	})
}

// [MOD-GF-MSG-2] network path.
func TestIncreaseActiveGovernanceFrameworkVersionNetwork(t *testing.T) {
	grace := types.DefaultGfAcceptanceGraceMin
	activate := func(signer string, grace time.Duration) *types.MsgIncreaseActiveGovernanceFrameworkVersion {
		return &types.MsgIncreaseActiveGovernanceFrameworkVersion{Signer: signer, Network: true, AcceptanceGrace: grace}
	}

	t.Run("only the council account signs it", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		_, err := keeper.NewMsgServerImpl(k).IncreaseActiveGovernanceFrameworkVersion(ctx, activate(testDelegate, grace))
		require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
	})

	t.Run("aborts without a draft", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		_, err := keeper.NewMsgServerImpl(k).IncreaseActiveGovernanceFrameworkVersion(ctx, activate(k.GetAuthority(), grace))
		require.ErrorIs(t, err, types.ErrNoActivatableVersion)
	})

	t.Run("aborts when acceptance_grace is under gf_acceptance_grace_min", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		addNetworkVersion(t, k, ctx, 2, 0)
		_, err := keeper.NewMsgServerImpl(k).IncreaseActiveGovernanceFrameworkVersion(ctx, activate(k.GetAuthority(), grace-time.Second))
		require.ErrorIs(t, err, types.ErrAcceptanceGraceTooShort)
	})

	t.Run("acceptance_grace is mandatory", func(t *testing.T) {
		require.ErrorIs(t, activate(testDelegate, 0).ValidateBasic(), types.ErrAcceptanceGraceTooShort)
	})

	t.Run("activates the draft and opens the grace period", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		addNetworkVersion(t, k, ctx, 2, 0)
		now := genesisTime.Add(time.Hour)
		ctx = ctx.WithBlockTime(now)
		_, err := keeper.NewMsgServerImpl(k).IncreaseActiveGovernanceFrameworkVersion(ctx, activate(k.GetAuthority(), 2*grace))
		require.NoError(t, err)

		gfv, err := k.GFVersion.Get(ctx, 2)
		require.NoError(t, err)
		require.Equal(t, now, *gfv.ActiveSince)
		ng, err := k.NetworkGovernance.Get(ctx)
		require.NoError(t, err)
		require.Equal(t, now.Add(2*grace), *ng.NgfAcceptanceDeadline)
		v, err := k.NetworkGFVersionInForce(ctx)
		require.NoError(t, err)
		require.Equal(t, uint32(2), v)

		attrs := eventAttrs(ctx, types.EventTypeIncreaseActiveGovernanceFrameworkVersion)
		require.Equal(t, "true", attrs[types.AttributeKeyNetwork])
		require.Equal(t, "2", attrs[types.AttributeKeyVersion])
		require.Equal(t, now.Format(time.RFC3339), attrs[types.AttributeKeyActiveSince])
		require.Equal(t, now.Add(2*grace).Format(time.RFC3339), attrs[types.AttributeKeyAcceptanceDeadline])
		require.Contains(t, attrs["msg"], `"acceptance_grace":"5184000s"`)
	})
}

// [MOD-GF-MSG-2] the network draft needs a document, whatever its language.
func TestIncreaseActiveNetworkVersionNeedsDocument(t *testing.T) {
	k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	ng, err := k.NetworkGovernance.Get(ctx)
	require.NoError(t, err)
	require.NoError(t, k.GFVersion.Set(ctx, 2, types.GovernanceFrameworkVersion{Id: 2, Version: 2}))
	ng.NetworkGovernanceFrameworkVersions = append(ng.NetworkGovernanceFrameworkVersions, 2)
	require.NoError(t, k.NetworkGovernance.Set(ctx, ng))

	_, err = keeper.NewMsgServerImpl(k).IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{
		Signer: k.GetAuthority(), Network: true, AcceptanceGrace: types.DefaultGfAcceptanceGraceMin,
	})
	require.ErrorIs(t, err, types.ErrMissingDefaultLang)
}

// [MOD-GF-MSG-2-3] a second activation during a grace replaces the deadline.
func TestIncreaseActiveNetworkVersionReplacesDeadline(t *testing.T) {
	grace := types.DefaultGfAcceptanceGraceMin
	k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
	addNetworkVersion(t, k, ctx, 2, grace)
	t1 := genesisTime.Add(time.Hour)
	addNetworkVersion(t, k, ctx.WithBlockTime(t1), 3, 2*grace)

	ng, err := k.NetworkGovernance.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, t1.Add(2*grace), *ng.NgfAcceptanceDeadline)
	co := types.CorporationView{Id: 1, AcceptedNgfVersion: 2}
	require.NoError(t, k.CheckNetworkGFAcceptance(ctx, co, genesisTime.Add(grace)))
	require.ErrorIs(t, k.CheckNetworkGFAcceptance(ctx, co, t1.Add(2*grace)), types.ErrGFNotAccepted)

	// A shorter grace shortens it too.
	t2 := t1.Add(time.Hour)
	addNetworkVersion(t, k, ctx.WithBlockTime(t2), 4, grace)
	ng, err = k.NetworkGovernance.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, t2.Add(grace), *ng.NgfAcceptanceDeadline)
}
