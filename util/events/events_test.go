package events_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/verana-labs/verana-node/util/events"
)

type role int

func (r role) String() string { return [...]string{"UNSPECIFIED", "ISSUER"}[r] }

func attrs(e sdk.Event) [][2]string {
	out := make([][2]string, 0, len(e.Attributes))
	for _, a := range e.Attributes {
		out = append(out, [2]string{a.Key, a.Value})
	}
	return out
}

func TestNewCarriesTypeAndModule(t *testing.T) {
	e := events.New("ac", "revoke_accreditation").Event()
	require.Equal(t, "revoke_accreditation", e.Type)
	require.Equal(t, [][2]string{{"module", "ac"}}, attrs(e))
}

func TestEncodings(t *testing.T) {
	now := time.Date(2026, 11, 24, 10, 30, 15, 987654321, time.FixedZone("CET", 3600))
	addr := sdk.AccAddress([]byte("addr________________"))

	e := events.New("td", "mint_trust_units").
		Uint("vs_id", 7).
		Str("vid", "did:web:example.com").
		Bool("exclude", false).
		Enum("role", role(1)).
		Dec("minted_tu", math.LegacyMustNewDecFromStr("1.5")).
		Int("amount", math.NewInt(1000000)).
		Coins("spend_limit", sdk.NewCoins(sdk.NewInt64Coin("uvna", 5), sdk.NewInt64Coin("uusd", 3))).
		Addr("grantee", addr).
		Bytes("digest", []byte{0xde, 0xad, 0xbe, 0xef}).
		Time("now", now).
		Duration("period", 90*time.Minute+500*time.Millisecond).
		Event()

	require.Equal(t, [][2]string{
		{"module", "td"},
		{"vs_id", "7"},
		{"vid", "did:web:example.com"},
		{"exclude", "false"},
		{"role", "ISSUER"},
		{"minted_tu", "1.500000000000000000"},
		{"amount", "1000000"},
		{"spend_limit", "3uusd,5uvna"},
		{"grantee", addr.String()},
		{"digest", "3q2+7w=="},
		{"now", "2026-11-24T09:30:15Z"},
		{"period", "5400"},
	}, attrs(e))
}

func TestAbsentValuesAreOmitted(t *testing.T) {
	e := events.New("ac", "self_create_accreditation").
		OptUint("validator_accreditation_id", nil).
		OptTime("effective_until", nil).
		Time("archived", time.Time{}).
		OptDuration("period", nil).
		Str("vid", "").
		Dec("rate", math.LegacyDec{}).
		Int("amount", math.Int{}).
		Coins("spend_limit", nil).
		Addr("grantee", nil).
		Bytes("digest", nil).
		Event()
	require.Equal(t, [][2]string{{"module", "ac"}}, attrs(e))
}

func TestOptionalValuesWhenSet(t *testing.T) {
	id := uint64(3)
	until := time.Unix(1795000000, 0)
	period := 30 * 24 * time.Hour
	e := events.New("ac", "self_create_accreditation").
		OptUint("validator_accreditation_id", &id).
		OptTime("effective_until", &until).
		OptDuration("period", &period).
		Event()
	require.Equal(t, [][2]string{
		{"module", "ac"},
		{"validator_accreditation_id", "3"},
		{"effective_until", "2026-11-18T11:06:40Z"},
		{"period", "2592000"},
	}, attrs(e))
}

func TestZeroValuesAreKept(t *testing.T) {
	e := events.New("td", "slash_trust_deposit").
		Uint("tu_amount", 0).
		Dec("fiat_amount", math.LegacyZeroDec()).
		Int("amount", math.ZeroInt()).
		Duration("transition", 0).
		Event()
	require.Equal(t, [][2]string{
		{"module", "td"},
		{"tu_amount", "0"},
		{"fiat_amount", "0.000000000000000000"},
		{"amount", "0"},
		{"transition", "0"},
	}, attrs(e))
}

func TestMsgCarriesProtoJSON(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	msg := &banktypes.MsgSend{
		FromAddress: "from",
		ToAddress:   "to",
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("uvna", 5)),
	}

	e := events.New("co", "set_sponsor")
	require.NoError(t, e.Msg(cdc, msg))
	require.Equal(t, [][2]string{
		{"module", "co"},
		{"msg", `{"from_address":"from","to_address":"to","amount":[{"denom":"uvna","amount":"5"}]}`},
	}, attrs(e.Event()))
}

func TestMsgReturnsMarshalError(t *testing.T) {
	cdc := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	msg := &authztypes.MsgExec{Grantee: "g", Msgs: []*codectypes.Any{{TypeUrl: "/unknown.Msg"}}}

	e := events.New("co", "set_sponsor")
	require.Error(t, e.Msg(cdc, msg))
	require.Equal(t, [][2]string{{"module", "co"}}, attrs(e.Event()))
}

func TestEmit(t *testing.T) {
	ctx := sdk.Context{}.WithEventManager(sdk.NewEventManager())
	events.New("di", "store_digest").Uint("vs_id", 1).Emit(ctx)

	evs := ctx.EventManager().Events()
	require.Len(t, evs, 1)
	require.Equal(t, "store_digest", evs[0].Type)
	require.Equal(t, [][2]string{{"module", "di"}, {"vs_id", "1"}}, attrs(evs[0]))
}
