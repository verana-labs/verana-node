package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/verana-labs/verana-node/testutil/keeper"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/gf/keeper"
	"github.com/verana-labs/verana-node/x/gf/types"
)

var genesisTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// eventAttrs returns the attributes of the last event of type typ.
func eventAttrs(ctx sdk.Context, typ string) map[string]string {
	attrs := map[string]string{}
	for _, ev := range ctx.EventManager().Events() {
		if ev.Type == typ {
			attrs = map[string]string{}
			for _, a := range ev.Attributes {
				attrs[a.Key] = a.Value
			}
		}
	}
	return attrs
}

// networkKeeper returns a keeper whose state is the default genesis: version 1
// of the network governance framework, active since genesisTime.
func networkKeeper(t *testing.T, del types.DelegationKeeper, eco *mockEcosystem, corp *mockCorporation) (keeper.Keeper, sdk.Context) {
	t.Helper()
	k, ctx := keepertest.GfKeeperWithDelegation(t, del, eco, corp)
	ctx = ctx.WithBlockTime(genesisTime)
	require.NoError(t, k.InitGenesis(ctx, *types.DefaultGenesis()))
	return k, ctx
}

// addNetworkVersion drafts the next network version with one document and,
// when grace is set, activates it at the block time.
func addNetworkVersion(t *testing.T, k keeper.Keeper, ctx sdk.Context, version uint32, grace time.Duration) {
	t.Helper()
	ms := keeper.NewMsgServerImpl(k)
	doc := validMsg("", "", 0, version)
	doc.Authority = k.GetAuthority()
	_, err := ms.AddGovernanceFrameworkDocument(ctx, doc)
	require.NoError(t, err)
	if grace > 0 {
		_, err = ms.IncreaseActiveGovernanceFrameworkVersion(ctx, &types.MsgIncreaseActiveGovernanceFrameworkVersion{Authority: k.GetAuthority(), AcceptanceGrace: grace})
		require.NoError(t, err)
	}
}

// [GF-CHECK-1]
func TestCheckNetworkGFAcceptance(t *testing.T) {
	grace := types.DefaultGfAcceptanceGraceMin

	t.Run("passes when the corporation accepted the version in force", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		require.NoError(t, k.CheckNetworkGFAcceptance(ctx, types.CorporationView{Id: 1, AcceptedNgfVersion: 1}, ctx.BlockTime()))
	})

	t.Run("aborts naming the version when nothing was accepted and no grace runs", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		err := k.CheckNetworkGFAcceptance(ctx, types.CorporationView{Id: 1}, ctx.BlockTime())
		require.ErrorIs(t, err, types.ErrGFNotAccepted)
		require.Contains(t, err.Error(), "version 1")
	})

	t.Run("a former version passes during the grace and fails from the deadline", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		addNetworkVersion(t, k, ctx, 2, grace)
		co := types.CorporationView{Id: 1, AcceptedNgfVersion: 1}

		require.NoError(t, k.CheckNetworkGFAcceptance(ctx, co, genesisTime.Add(grace-time.Second)))
		err := k.CheckNetworkGFAcceptance(ctx, co, genesisTime.Add(grace))
		require.ErrorIs(t, err, types.ErrGFNotAccepted)
		require.Contains(t, err.Error(), "version 2")

		co.AcceptedNgfVersion = 2
		require.NoError(t, k.CheckNetworkGFAcceptance(ctx, co, genesisTime.Add(grace)))
	})

	t.Run("a draft is not the version in force", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		addNetworkVersion(t, k, ctx, 2, 0)
		require.NoError(t, k.CheckNetworkGFAcceptance(ctx, types.CorporationView{Id: 1, AcceptedNgfVersion: 1}, ctx.BlockTime()))
	})

	t.Run("aborts when no network version is in force", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		require.ErrorIs(t, k.CheckNetworkGFAcceptance(ctx, types.CorporationView{Id: 1}, ctx.BlockTime()), types.ErrSubjectNotFound)
	})
}

// [GF-CHECK-2]
func TestCheckDelegateCGFAcceptance(t *testing.T) {
	now := genesisTime
	deadline := now.Add(time.Hour)
	tests := []struct {
		name     string
		co       types.CorporationView
		accepted uint32
		err      error
	}{
		{"accepted the active version", types.CorporationView{Id: 1, ActiveVersion: 2}, 2, nil},
		{"never accepted, even during a grace", types.CorporationView{Id: 1, ActiveVersion: 2, AcceptanceDeadline: &deadline}, 0, types.ErrGFNotAccepted},
		{"former version during the grace", types.CorporationView{Id: 1, ActiveVersion: 2, AcceptanceDeadline: &deadline}, 1, nil},
		{"former version without a grace", types.CorporationView{Id: 1, ActiveVersion: 2}, 1, types.ErrGFNotAccepted},
		{"former version at the deadline", types.CorporationView{Id: 1, ActiveVersion: 2, AcceptanceDeadline: &now}, 1, types.ErrGFNotAccepted},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{view: tc.co, found: true})
			err := k.CheckDelegateCGFAcceptance(ctx, 1, &detypes.DelegateAuthorization{Delegate: testDelegate, AcceptedCgfVersion: tc.accepted}, now)
			if tc.err == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tc.err)
			require.Contains(t, err.Error(), "version 2")
		})
	}

	t.Run("aborts when the corporation does not exist", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		require.ErrorIs(t, k.CheckDelegateCGFAcceptance(ctx, 1, &detypes.DelegateAuthorization{}, now), types.ErrSubjectNotFound)
	})
}

// [GF-CHECK-3]
func TestCheckEcosystemGFAcceptance(t *testing.T) {
	now := genesisTime
	deadline := now.Add(time.Hour)
	eco := func(deadline *time.Time) *mockEcosystem {
		return &mockEcosystem{view: types.EcosystemView{Id: 7, CorporationID: 9, ActiveVersion: 2, AcceptanceDeadline: deadline}, found: true}
	}
	accept := func(t *testing.T, e *mockEcosystem, version uint32) (keeper.Keeper, sdk.Context) {
		t.Helper()
		e.view.ActiveVersion = version
		corp := &mockCorporation{view: types.CorporationView{Id: 1, PolicyAddress: testCorp}, found: true}
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, e, corp)
		_, err := keeper.NewMsgServerImpl(k).AcceptGovernanceFrameworkVersion(ctx, &types.MsgAcceptGovernanceFrameworkVersion{
			Corporation: testCorp, Delegate: testDelegate, EcosystemId: 7, Version: version,
		})
		require.NoError(t, err)
		e.view.ActiveVersion = 2
		return k, ctx
	}

	t.Run("the controller acts without accepting", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco(nil), &mockCorporation{})
		require.NoError(t, k.CheckEcosystemGFAcceptance(ctx, 7, 9, now))
	})
	t.Run("aborts naming the version when the corporation never accepted", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, eco(&deadline), &mockCorporation{})
		err := k.CheckEcosystemGFAcceptance(ctx, 7, 1, now)
		require.ErrorIs(t, err, types.ErrGFNotAccepted)
		require.Contains(t, err.Error(), "version 2")
	})
	t.Run("passes on the active version", func(t *testing.T) {
		k, ctx := accept(t, eco(nil), 2)
		require.NoError(t, k.CheckEcosystemGFAcceptance(ctx, 7, 1, now))
	})
	t.Run("a former version passes during the grace only", func(t *testing.T) {
		k, ctx := accept(t, eco(&deadline), 1)
		require.NoError(t, k.CheckEcosystemGFAcceptance(ctx, 7, 1, now))
		require.ErrorIs(t, k.CheckEcosystemGFAcceptance(ctx, 7, 1, deadline), types.ErrGFNotAccepted)
	})
	t.Run("a former version fails without a grace", func(t *testing.T) {
		k, ctx := accept(t, eco(nil), 1)
		require.ErrorIs(t, k.CheckEcosystemGFAcceptance(ctx, 7, 1, now), types.ErrGFNotAccepted)
	})
	t.Run("aborts when the ecosystem does not exist", func(t *testing.T) {
		k, ctx := keepertest.GfKeeperWithDelegation(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		require.ErrorIs(t, k.CheckEcosystemGFAcceptance(ctx, 7, 1, now), types.ErrSubjectNotFound)
	})
}
