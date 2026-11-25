// Package events builds the events through which indexers rebuild the history
// of the registry.
//
// Convention:
//   - the type is the spec method name in snake_case, and every event carries
//     the emitting module in "module";
//   - attribute names are the spec field names, block time is "now", and the
//     parameters of a method executed through a proposal are carried as the
//     proto JSON of its message in "msg";
//   - values use their API form: decimal integers, RFC 3339 UTC timestamps at
//     second precision, 18-digit decimals, bech32 accounts, enum names, base64
//     bytes, durations as seconds;
//   - an absent value (nil, empty or a zero time) is omitted, a zero number or
//     duration is kept.
package events

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/math"
	"github.com/cosmos/gogoproto/proto"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Event struct {
	ev sdk.Event
}

func New(module, eventType string) *Event {
	return &Event{ev: sdk.NewEvent(eventType, sdk.NewAttribute(AttrModule, module))}
}

func (e *Event) add(key, value string) *Event {
	e.ev = e.ev.AppendAttributes(sdk.NewAttribute(key, value))
	return e
}

func (e *Event) Uint(key string, v uint64) *Event {
	return e.add(key, strconv.FormatUint(v, 10))
}

func (e *Event) OptUint(key string, v *uint64) *Event {
	if v == nil {
		return e
	}
	return e.Uint(key, *v)
}

func (e *Event) Str(key, v string) *Event {
	if v == "" {
		return e
	}
	return e.add(key, v)
}

func (e *Event) Bool(key string, v bool) *Event {
	return e.add(key, strconv.FormatBool(v))
}

func (e *Event) Enum(key string, v fmt.Stringer) *Event {
	return e.add(key, v.String())
}

func (e *Event) Time(key string, t time.Time) *Event {
	if t.IsZero() {
		return e
	}
	return e.add(key, t.UTC().Format(time.RFC3339))
}

func (e *Event) OptTime(key string, t *time.Time) *Event {
	if t == nil {
		return e
	}
	return e.Time(key, *t)
}

func (e *Event) Duration(key string, d time.Duration) *Event {
	return e.add(key, strconv.FormatInt(int64(d/time.Second), 10))
}

func (e *Event) OptDuration(key string, d *time.Duration) *Event {
	if d == nil {
		return e
	}
	return e.Duration(key, *d)
}

func (e *Event) Dec(key string, v math.LegacyDec) *Event {
	if v.IsNil() {
		return e
	}
	return e.add(key, v.String())
}

func (e *Event) Int(key string, v math.Int) *Event {
	if v.IsNil() {
		return e
	}
	return e.add(key, v.String())
}

func (e *Event) Coins(key string, v sdk.Coins) *Event {
	if v.Empty() {
		return e
	}
	return e.add(key, v.String())
}

func (e *Event) Addr(key string, v sdk.AccAddress) *Event {
	if v.Empty() {
		return e
	}
	return e.add(key, v.String())
}

func (e *Event) Bytes(key string, v []byte) *Event {
	if len(v) == 0 {
		return e
	}
	return e.add(key, base64.StdEncoding.EncodeToString(v))
}

// Msg adds the parameters of a method executed through a proposal.
func (e *Event) Msg(cdc codec.JSONCodec, msg proto.Message) error {
	bz, err := cdc.MarshalJSON(msg)
	if err != nil {
		return err
	}
	e.add(AttrMsg, string(bz))
	return nil
}

func (e *Event) Event() sdk.Event {
	return e.ev
}

func (e *Event) Emit(ctx sdk.Context) {
	ctx.EventManager().EmitEvent(e.ev)
}
