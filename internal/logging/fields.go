package logging

import (
	types "code.vegaprotocol.io/vega/proto"

	"go.uber.org/zap"
)

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

// Error constructs a field with the given error value.
func Error(val error) zap.Field {
	return zap.Error(val)
}

// AccountType constructs a field with the given VEGA market proto value.
func AccountType(at types.AccountType) zap.Field {
	return zap.String("account-type", at.String())
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

// OrderID constructs a field with the given VEGA market proto value.
func OrderID(id string) zap.Field {
	return zap.String("order-id", id)
}

// OrderWithTag constructs a field with the given VEGA order proto value and key equal to the tag string.
func OrderWithTag(o types.Order, tag string) zap.Field {
	return zap.String(tag, o.String())
}

// PendingOrder constructs a field with the given VEGA order proto value.
func PendingOrder(po types.PendingOrder) zap.Field {
	return zap.String("pending-order", po.String())
}

// Trade constructs a field with the given VEGA trade proto value.
func Trade(t types.Trade) zap.Field {
	return zap.String("trade", t.String())
}

// Market constructs a field with the given VEGA market proto value.
func Market(m types.Market) zap.Field {
	return zap.String("market", m.String())
}

// MarketID constructs a field with the given VEGA market proto value.
func MarketID(id string) zap.Field {
	return zap.String("market-id", id)
}

// Party constructs a field with the given VEGA party proto value.
func Party(p types.Party) zap.Field {
	return zap.String("party", p.String())
}

// PartyID constructs a field with the given VEGA market proto value.
func PartyID(id string) zap.Field {
	return zap.String("party-id", id)
}

func Reflect(key string, val interface{}) zap.Field {
	return zap.Reflect(key, val)
}
