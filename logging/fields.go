package logging

import (
	"encoding/hex"
	"time"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"

	"go.uber.org/zap"
)

func Hash(h []byte) zap.Field {
	hs := hex.EncodeToString(h)
	return zap.String("hash", hs)
}

// Binary constructs a field that carries an opaque binary blob.
//
// Binary data is serialized in an encoding-appropriate format. For example,
// zap's JSON encoder base64-encodes binary blobs. To log UTF-8 encoded text,
// use ByteString.
func Binary(key string, val []byte) zap.Field {
	return zap.Binary(key, val)
}

// Bool constructs a field that carries a bool.
func Bool(key string, val bool) zap.Field {
	return zap.Bool(key, val)
}

// ByteString constructs a field that carries UTF-8 encoded text as a []byte.
// To log opaque binary blobs (which aren't necessarily valid UTF-8), use
// Binary.
func ByteString(key string, val []byte) zap.Field {
	return zap.ByteString(key, val)
}

// Complex128 constructs a field that carries a complex number. Unlike most
// numeric fields, this costs an allocation (to convert the complex128 to
// interface{}).
func Complex128(key string, val complex128) zap.Field {
	return zap.Complex128(key, val)
}

// Complex64 constructs a field that carries a complex number. Unlike most
// numeric fields, this costs an allocation (to convert the complex64 to
// interface{}).
func Complex64(key string, val complex64) zap.Field {
	return zap.Complex64(key, val)
}

// Float64 constructs a field that carries a float64. The way the
// floating-point value is represented is encoder-dependent, so marshaling is
// necessarily lazy.
func Float64(key string, val float64) zap.Field {
	return zap.Float64(key, val)
}

// Float32 constructs a field that carries a float32. The way the
// floating-point value is represented is encoder-dependent, so marshaling is
// necessarily lazy.
func Float32(key string, val float32) zap.Field {
	return zap.Float32(key, val)
}

// Int constructs a field with the given key and value.
func Int(key string, val int) zap.Field {
	return Int64(key, int64(val))
}

// Int64 constructs a field with the given key and value.
func Int64(key string, val int64) zap.Field {
	return zap.Int64(key, val)
}

// Int32 constructs a field with the given key and value.
func Int32(key string, val int32) zap.Field {
	return zap.Int32(key, val)
}

// Int16 constructs a field with the given key and value.
func Int16(key string, val int16) zap.Field {
	return zap.Int16(key, val)
}

// Int8 constructs a field with the given key and value.
func Int8(key string, val int8) zap.Field {
	return zap.Int8(key, val)
}

// String constructs a field with the given key and value.
func String(key string, val string) zap.Field {
	return zap.String(key, val)
}

// Uint constructs a field with the given key and value.
func Uint(key string, val uint) zap.Field {
	return Uint64(key, uint64(val))
}

// Uint64 constructs a field with the given key and value.
func Uint64(key string, val uint64) zap.Field {
	return zap.Uint64(key, val)
}

// Uint32 constructs a field with the given key and value.
func Uint32(key string, val uint32) zap.Field {
	return zap.Uint32(key, val)
}

// Uint16 constructs a field with the given key and value.
func Uint16(key string, val uint16) zap.Field {
	return zap.Uint16(key, val)
}

// Uint8 constructs a field with the given key and value.
func Uint8(key string, val uint8) zap.Field {
	return zap.Uint8(key, val)
}

// Uintptr constructs a field with the given key and value.
func Uintptr(key string, val uintptr) zap.Field {
	return zap.Uintptr(key, val)
}

func Duration(key string, value time.Duration) zap.Field {
	return zap.String(key, value.String())
}

// Error constructs a field with the given error value.
func Error(val error) zap.Field {
	return zap.Error(val)
}

// Candle constructs a field with the given VEGA candle proto value.
func Candle(c types.Candle) zap.Field {
	return zap.String("candle", c.String())
}

// CandleWithTag constructs a field with the given VEGA candle proto value and key equal to the tag string.
func CandleWithTag(c types.Candle, tag string) zap.Field {
	return zap.String(tag, c.String())
}

// Order constructs a field with the given VEGA order proto value.
func Order(o types.Order) zap.Field {
	return zap.String("order", o.String())
}

func OrderID(id string) zap.Field {
	return zap.String("order-id", id)
}

// Time display a time
func Time(key string, t time.Time) zap.Field {
	return zap.Time(key, t)
}

// OrderWithTag constructs a field with the given VEGA order proto value and key equal to the tag string.
func OrderWithTag(o types.Order, tag string) zap.Field {
	return zap.String(tag, o.String())
}

// Trade constructs a field with the given VEGA trade proto value.
func Trade(t types.Trade) zap.Field {
	return zap.String("trade", t.String())
}

// Market constructs a field with the given VEGA market proto value.
func Market(m types.Market) zap.Field {
	return zap.String("market", m.String())
}

func MarketID(id string) zap.Field {
	return zap.String("market-id", id)
}

func AssetID(id string) zap.Field {
	return zap.String("asset-id", id)
}

func LiquidityID(id string) zap.Field {
	return zap.String("liquidity-id", id)
}

func LiquidityProvisionSubmission(lp types.LiquidityProvisionSubmission) zap.Field {
	return zap.String("liquidity-provision-submission", lp.String())
}

// Party constructs a field with the given VEGA party proto value.
func Party(p types.Party) zap.Field {
	return zap.String("party", p.String())
}

func PartyID(id string) zap.Field {
	return zap.String("party", id)
}

// Account constructs a field with the given VEGA account proto value.
func Account(a types.Account) zap.Field {
	return zap.String("account", a.String())
}

// OrderAmendment constructs a single string field to contain all the object information
func OrderAmendment(oa *types.OrderAmendment) zap.Field {
	return zap.String("order-amendment", oa.String())
}

func OrderCancellation(oc *types.OrderCancellation) zap.Field {
	return zap.String("order-cancellation", oc.String())
}

// Reflect constructs a field by running reflection over all the
// field of value passed as a parameter.
// This should never be used we basic log level,
// only in the case of Debuf log level, and with guards on  top
// of the actual log call for this level
func Reflect(key string, val interface{}) zap.Field {
	return zap.Reflect(key, val)
}

// TraceID logs the event traceID
func TraceID(e events.Event) zap.Field {
	return zap.String("trace-id", e.TraceID())
}

// EventType logs the event type as a string
func EventType(e events.Event) zap.Field {
	return zap.String("event-type", e.Type().String())
}
