package statevar

import (
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
)

// value is an interface for representing differnet types of floating point scalars vectors and matrices.
type value interface {
	Equals(other value) bool
	WithinTolerance(other value, tolerance num.Decimal) bool
	ToProto() *vega.StateVarValue
}

// KeyValueBundle is a slice of key value and their expected tolerances.
type KeyValueBundle struct {
	KVT []KeyValueTol
}

// ToProto converts KevValueBundle into proto.
func (kvb *KeyValueBundle) ToProto() []*vega.KeyValueBundle {
	res := make([]*vega.KeyValueBundle, 0, len(kvb.KVT))
	for _, kvt := range kvb.KVT {
		res = append(res, &vega.KeyValueBundle{
			Key:       kvt.Key,
			Tolerance: kvt.Tolerance.String(),
			Value:     kvt.Val.ToProto(),
		})
	}
	return res
}

// KeyValueBundleFromProto converts from proto into KeyValueBundle.
func KeyValueBundleFromProto(protoKVT []*vega.KeyValueBundle) *KeyValueBundle {
	KVT := make([]KeyValueTol, 0, len(protoKVT))
	for _, pKVT := range protoKVT {
		tol, err := num.DecimalFromString(pKVT.Tolerance)
		if err != nil {
			continue
		}
		KVT = append(KVT, KeyValueTol{
			Key:       pKVT.Key,
			Tolerance: tol,
			Val:       ValueFromProto(pKVT.Value),
		})
	}
	return &KeyValueBundle{
		KVT: KVT,
	}
}

// ValueFromProto converts the proto into a value.
func ValueFromProto(val *vega.StateVarValue) value {
	switch v := val.Value.(type) {
	case *vega.StateVarValue_ScalarVal:
		val, _ := num.DecimalFromString(v.ScalarVal.Value)
		return &DecimalScalar{
			Val: val,
		}
	case *vega.StateVarValue_VectorVal:
		vec := make([]num.Decimal, 0, len(v.VectorVal.Value))
		for _, entry := range v.VectorVal.Value {
			value, err := num.DecimalFromString(entry)
			if err == nil {
				vec = append(vec, value)
			}
		}

		return &DecimalVector{
			Val: vec,
		}
	case *vega.StateVarValue_MatrixVal:
		mat := [][]num.Decimal{}
		for _, val := range v.MatrixVal.Value {
			row := make([]num.Decimal, 0, len(val.Value))
			for _, entry := range val.Value {
				value, err := num.DecimalFromString(entry)
				if err == nil {
					row = append(row, value)
				}
			}
			mat = append(mat, row)
		}
		return &DecimalMatrix{
			Val: mat,
		}
	default:
		return nil
	}
}

// WithinTolerance returns true if the two bundles have the same keys, same tolerances and the values at the same index are with the tolerance of each other.
func (kvb *KeyValueBundle) WithinTolerance(other *KeyValueBundle) bool {
	if len(kvb.KVT) != len(other.KVT) {
		return false
	}
	for i, kv := range kvb.KVT {
		if kv.Key != other.KVT[i].Key {
			return false
		}
		if !kv.Tolerance.Equal(other.KVT[i].Tolerance) {
			return false
		}

		if !kv.Val.WithinTolerance(other.KVT[i].Val, kv.Tolerance) {
			return false
		}
	}
	return true
}

// Equals returns true of the two bundles have the same keys in the same order and the values in the same index are equal.
func (kvb *KeyValueBundle) Equals(other *KeyValueBundle) bool {
	if len(kvb.KVT) != len(other.KVT) {
		return false
	}
	for i, kv := range kvb.KVT {
		if kv.Key != other.KVT[i].Key {
			return false
		}
		if !kv.Val.Equals(other.KVT[i].Val) {
			return false
		}
	}
	return true
}

type KeyValueTol struct {
	Key       string      // the name of the key
	Val       value       // the floating point value (scalar, vector, matrix)
	Tolerance num.Decimal // the tolerance to use in comparison
}

type DecimalValue interface{}

type FinaliseCalculation interface {
	CalculationFinished(string, StateVariableResult, error)
}

type StateVariableResult interface{}

type Converter interface {
	BundleToInterface(*KeyValueBundle) StateVariableResult
	InterfaceToBundle(StateVariableResult) *KeyValueBundle
}

// StateVarEventType enumeration for supported events triggering calculation.
type StateVarEventType int

const (
	// sample events there may be many more.

	StateVarEventTypeAuctionUnknown StateVarEventType = iota
	StateVarEventTypeAuctionEnded
	StateVarEventMarketEnactment
	StateVarEventTypeMarketEnacatment
	StateVarEventTypeRiskModelChanged
)
