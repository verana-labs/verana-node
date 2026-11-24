package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	cotypes "github.com/verana-labs/verana-node/x/co/types"
	detypes "github.com/verana-labs/verana-node/x/de/types"
	"github.com/verana-labs/verana-node/x/gf/keeper"
	"github.com/verana-labs/verana-node/x/gf/types"
)

func acceptMsg(delegate string, ecosystemID uint64, network bool, version uint32) *types.MsgAcceptGovernanceFrameworkVersion {
	return &types.MsgAcceptGovernanceFrameworkVersion{Corporation: testCorp, Delegate: delegate, EcosystemId: ecosystemID, Network: network, Version: version}
}

func TestMsgAcceptGovernanceFrameworkVersionValidateBasic(t *testing.T) {
	require.NoError(t, acceptMsg(testDelegate, 7, false, 1).ValidateBasic())
	require.NoError(t, acceptMsg(testDelegate, 0, true, 1).ValidateBasic())
	require.ErrorIs(t, acceptMsg(testDelegate, 7, true, 1).ValidateBasic(), types.ErrInvalidSubject)
	require.ErrorIs(t, acceptMsg(testDelegate, 0, false, 1).ValidateBasic(), types.ErrInvalidSubject)
	require.ErrorIs(t, acceptMsg(testDelegate, 7, false, 0).ValidateBasic(), types.ErrInvalidVersion)
	require.Error(t, acceptMsg("bad", 7, false, 1).ValidateBasic())
}

// [MOD-GF-MSG-3] with network true.
func TestAcceptNetworkGovernanceFramework(t *testing.T) {
	corp := func() *mockCorporation {
		return &mockCorporation{view: types.CorporationView{Id: 1, PolicyAddress: testCorp}, found: true, gfCheck1Err: types.ErrGFNotAccepted}
	}

	t.Run("a delegate renews the acceptance without GF-CHECK-1", func(t *testing.T) {
		del := &recordingDelegation{}
		c := corp()
		k, ctx := networkKeeper(t, del, &mockEcosystem{}, c)
		_, err := keeper.NewMsgServerImpl(k).AcceptGovernanceFrameworkVersion(ctx, acceptMsg(testDelegate, 0, true, 1))
		require.NoError(t, err)
		require.Equal(t, 1, del.calls)
		require.Equal(t, detypes.MsgType_MSG_TYPE_ACCEPT_GOVERNANCE_FRAMEWORK_VERSION, del.gotMsgType)
		require.Equal(t, uint32(1), c.acceptedNgf)

		attrs := eventAttrs(ctx, types.EventTypeAcceptGovernanceFrameworkVersion)
		require.Equal(t, map[string]string{
			"module":                  types.ModuleName,
			"corporation_id":          "1",
			types.AttributeKeyNetwork: "true",
			types.AttributeKeyVersion: "1",
			"now":                     genesisTime.Format(time.RFC3339),
		}, attrs)
	})

	t.Run("the group path needs no delegate authorization and carries the message", func(t *testing.T) {
		del := &recordingDelegation{}
		c := corp()
		k, ctx := networkKeeper(t, del, &mockEcosystem{}, c)
		_, err := keeper.NewMsgServerImpl(k).AcceptGovernanceFrameworkVersion(ctx, acceptMsg(testCorp, 0, true, 1))
		require.NoError(t, err)
		require.Zero(t, del.calls)
		require.Equal(t, uint32(1), c.acceptedNgf)
		require.Contains(t, eventAttrs(ctx, types.EventTypeAcceptGovernanceFrameworkVersion)["msg"], `"network":true`)
	})

	t.Run("only the version in force can be accepted", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, corp())
		addNetworkVersion(t, k, ctx, 2, 0)
		_, err := keeper.NewMsgServerImpl(k).AcceptGovernanceFrameworkVersion(ctx, acceptMsg(testDelegate, 0, true, 2))
		require.ErrorIs(t, err, types.ErrInvalidVersion)
	})

	t.Run("a former version cannot be accepted", func(t *testing.T) {
		c := corp()
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, c)
		addNetworkVersion(t, k, ctx, 2, types.DefaultGfAcceptanceGraceMin)
		_, err := keeper.NewMsgServerImpl(k).AcceptGovernanceFrameworkVersion(ctx, acceptMsg(testDelegate, 0, true, 1))
		require.ErrorIs(t, err, types.ErrInvalidVersion)
		require.Zero(t, c.acceptedNgf)
	})

	t.Run("aborts for an unregistered corporation", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, &mockCorporation{})
		_, err := keeper.NewMsgServerImpl(k).AcceptGovernanceFrameworkVersion(ctx, acceptMsg(testDelegate, 0, true, 1))
		require.ErrorIs(t, err, cotypes.ErrCorporationNotRegistered)
	})

	t.Run("aborts when AUTHZ-CHECK-1 fails", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{err: detypes.ErrAuthzNotFound}, &mockEcosystem{}, corp())
		_, err := keeper.NewMsgServerImpl(k).AcceptGovernanceFrameworkVersion(ctx, acceptMsg(testDelegate, 0, true, 1))
		require.ErrorIs(t, err, detypes.ErrAuthzNotFound)
	})
}

// [MOD-GF-MSG-3] with ecosystem_id.
func TestAcceptEcosystemGovernanceFramework(t *testing.T) {
	corp := &mockCorporation{view: types.CorporationView{Id: 1, PolicyAddress: testCorp}, found: true}
	eco := func(controller uint64) *mockEcosystem {
		return &mockEcosystem{view: types.EcosystemView{Id: 7, CorporationID: controller, ActiveVersion: 3}, found: true}
	}

	t.Run("records the acceptance and serves it", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, eco(9), corp)
		ms := keeper.NewMsgServerImpl(k)
		_, err := ms.AcceptGovernanceFrameworkVersion(ctx, acceptMsg(testDelegate, 7, false, 3))
		require.NoError(t, err)

		later := genesisTime.Add(time.Hour)
		_, err = ms.AcceptGovernanceFrameworkVersion(ctx.WithBlockTime(later), acceptMsg(testDelegate, 7, false, 3))
		require.NoError(t, err)

		resp, err := keeper.NewQueryServerImpl(k).GetEcosystemAcceptance(ctx, &types.QueryGetEcosystemAcceptanceRequest{EcosystemId: 7, CorporationId: 1})
		require.NoError(t, err)
		require.Equal(t, types.EcosystemAcceptance{EcosystemId: 7, CorporationId: 1, Version: 3, Accepted: later}, resp.Acceptance)

		stored, err := k.EcosystemAcceptances.Get(ctx, collections.Join(uint64(7), uint64(1)))
		require.NoError(t, err)
		require.Zero(t, stored.EcosystemId, "key fields are not stored in the value")
		require.Zero(t, stored.CorporationId)

		attrs := eventAttrs(ctx, types.EventTypeAcceptGovernanceFrameworkVersion)
		require.Equal(t, "7", attrs["ecosystem_id"])
		require.NotContains(t, attrs, types.AttributeKeyNetwork)
		require.NotContains(t, attrs, "msg")
	})

	t.Run("GF-CHECK-1 applies on both paths", func(t *testing.T) {
		c := &mockCorporation{view: corp.view, found: true, gfCheck1Err: types.ErrGFNotAccepted}
		k, ctx := networkKeeper(t, mockDelegation{}, eco(9), c)
		for _, delegate := range []string{testDelegate, testCorp} {
			_, err := keeper.NewMsgServerImpl(k).AcceptGovernanceFrameworkVersion(ctx, acceptMsg(delegate, 7, false, 3))
			require.ErrorIs(t, err, types.ErrGFNotAccepted)
		}
	})

	t.Run("aborts for an unknown ecosystem", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, &mockEcosystem{}, corp)
		_, err := keeper.NewMsgServerImpl(k).AcceptGovernanceFrameworkVersion(ctx, acceptMsg(testDelegate, 7, false, 3))
		require.ErrorIs(t, err, types.ErrSubjectNotFound)
	})

	t.Run("the controller accepts nothing", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, eco(1), corp)
		_, err := keeper.NewMsgServerImpl(k).AcceptGovernanceFrameworkVersion(ctx, acceptMsg(testDelegate, 7, false, 3))
		require.ErrorIs(t, err, types.ErrControllerAcceptance)
	})

	t.Run("only the active version can be accepted", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, eco(9), corp)
		for _, v := range []uint32{2, 4} {
			_, err := keeper.NewMsgServerImpl(k).AcceptGovernanceFrameworkVersion(ctx, acceptMsg(testDelegate, 7, false, v))
			require.ErrorIs(t, err, types.ErrInvalidVersion)
		}
		_, err := keeper.NewQueryServerImpl(k).GetEcosystemAcceptance(ctx, &types.QueryGetEcosystemAcceptanceRequest{EcosystemId: 7, CorporationId: 1})
		require.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("Get Ecosystem Acceptance returns not found", func(t *testing.T) {
		k, ctx := networkKeeper(t, mockDelegation{}, eco(9), corp)
		_, err := keeper.NewQueryServerImpl(k).GetEcosystemAcceptance(ctx, &types.QueryGetEcosystemAcceptanceRequest{EcosystemId: 7, CorporationId: 1})
		require.Equal(t, codes.NotFound, status.Code(err))
	})
}

// [MOD-GF-MSG-4]
func TestAcceptCorporationGovernanceFramework(t *testing.T) {
	msg := func(version uint32) *types.MsgAcceptCorporationGovernanceFramework {
		return &types.MsgAcceptCorporationGovernanceFramework{Delegate: testDelegate, CorporationId: 1, Version: version}
	}
	corp := &mockCorporation{view: types.CorporationView{Id: 1, ActiveVersion: 2}, found: true}

	t.Run("validate basic", func(t *testing.T) {
		require.NoError(t, msg(1).ValidateBasic())
		require.ErrorIs(t, msg(0).ValidateBasic(), types.ErrInvalidVersion)
		require.ErrorIs(t, (&types.MsgAcceptCorporationGovernanceFramework{Delegate: testDelegate, Version: 1}).ValidateBasic(), types.ErrInvalidSubject)
		require.Error(t, (&types.MsgAcceptCorporationGovernanceFramework{Delegate: "bad", CorporationId: 1, Version: 1}).ValidateBasic())
	})

	t.Run("records the delegate's acceptance", func(t *testing.T) {
		del := &recordingDelegation{}
		k, ctx := networkKeeper(t, del, &mockEcosystem{}, corp)
		_, err := keeper.NewMsgServerImpl(k).AcceptCorporationGovernanceFramework(ctx, msg(2))
		require.NoError(t, err)
		require.Equal(t, uint32(2), del.accepted)
		require.Equal(t, testDelegate, del.acceptedBy)
		require.Equal(t, uint64(1), del.acceptedFor)
		require.Zero(t, del.calls, "outside AUTHZ-CHECK")

		require.Equal(t, map[string]string{
			"module":                   types.ModuleName,
			"corporation_id":           "1",
			types.AttributeKeyDelegate: testDelegate,
			types.AttributeKeyVersion:  "2",
			"now":                      genesisTime.Format(time.RFC3339),
		}, eventAttrs(ctx, types.EventTypeAcceptCorporationGovernanceFramework))
	})

	t.Run("aborts for an unknown corporation", func(t *testing.T) {
		k, ctx := networkKeeper(t, &recordingDelegation{}, &mockEcosystem{}, &mockCorporation{})
		_, err := keeper.NewMsgServerImpl(k).AcceptCorporationGovernanceFramework(ctx, msg(2))
		require.ErrorIs(t, err, types.ErrSubjectNotFound)
	})

	t.Run("aborts without a delegate authorization", func(t *testing.T) {
		k, ctx := networkKeeper(t, &recordingDelegation{acceptErr: detypes.ErrAuthzNotFound}, &mockEcosystem{}, corp)
		_, err := keeper.NewMsgServerImpl(k).AcceptCorporationGovernanceFramework(ctx, msg(2))
		require.ErrorIs(t, err, detypes.ErrAuthzNotFound)
	})

	t.Run("only the active version can be accepted", func(t *testing.T) {
		del := &recordingDelegation{}
		k, ctx := networkKeeper(t, del, &mockEcosystem{}, corp)
		for _, v := range []uint32{1, 3} {
			_, err := keeper.NewMsgServerImpl(k).AcceptCorporationGovernanceFramework(ctx, msg(v))
			require.ErrorIs(t, err, types.ErrInvalidVersion)
		}
		require.Zero(t, del.accepted)
	})
}
