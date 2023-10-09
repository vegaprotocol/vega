// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package statevar

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
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
func (kvb KeyValueBundle) ToProto() []*vega.KeyValueBundle {
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
func KeyValueBundleFromProto(protoKVT []*vega.KeyValueBundle) (*KeyValueBundle, error) {
	KVT := make([]KeyValueTol, 0, len(protoKVT))
	for _, pKVT := range protoKVT {
		tol, err := num.DecimalFromString(pKVT.Tolerance)
		if err != nil {
			return nil, err
		}

		v, err := ValueFromProto(pKVT.Value)
		if err != nil {
			return nil, err
		}
		KVT = append(KVT, KeyValueTol{
			Key:       pKVT.Key,
			Tolerance: tol,
			Val:       v,
		})
	}
	return &KeyValueBundle{
		KVT: KVT,
	}, nil
}

// ValueFromProto converts the proto into a value.
func ValueFromProto(val *vega.StateVarValue) (value, error) {
	if val == nil {
		return nil, fmt.Errorf("missing state-var value")
	}
	switch v := val.Value.(type) {
	case *vega.StateVarValue_ScalarVal:
		val, err := num.DecimalFromString(v.ScalarVal.Value)
		if err != nil {
			return nil, err
		}
		return &DecimalScalar{
			Val: val,
		}, nil
	case *vega.StateVarValue_VectorVal:
		vec := make([]num.Decimal, 0, len(v.VectorVal.Value))
		for _, entry := range v.VectorVal.Value {
			value, err := num.DecimalFromString(entry)
			if err != nil {
				return nil, err
			}
			vec = append(vec, value)
		}
		return &DecimalVector{
			Val: vec,
		}, nil
	case *vega.StateVarValue_MatrixVal:
		mat := make([][]num.Decimal, 0, len(v.MatrixVal.Value))
		for _, val := range v.MatrixVal.Value {
			row := make([]num.Decimal, 0, len(val.Value))
			for _, entry := range val.Value {
				value, err := num.DecimalFromString(entry)
				if err != nil {
					return nil, err
				}
				row = append(row, value)
			}
			mat = append(mat, row)
		}
		return &DecimalMatrix{
			Val: mat,
		}, nil
	default:
		return nil, nil
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

type FinaliseCalculation interface {
	CalculationFinished(string, StateVariableResult, error)
}

type StateVariableResult interface{}

type Converter interface {
	BundleToInterface(*KeyValueBundle) StateVariableResult
	InterfaceToBundle(StateVariableResult) *KeyValueBundle
}

// EventType enumeration for supported events triggering calculation.
type EventType int

const (
	// sample events there may be many more.

	EventTypeAuctionUnknown EventType = iota
	EventTypeMarketEnactment
	EventTypeOpeningAuctionFirstUncrossingPrice
	EventTypeAuctionEnded
	EventTypeTimeTrigger
	EventTypeMarketUpdated
)

var StateVarEventTypeToName = map[EventType]string{
	EventTypeAuctionUnknown:                     "unknown",
	EventTypeMarketEnactment:                    "market-enacted",
	EventTypeOpeningAuctionFirstUncrossingPrice: "opening-auction-first-uncrossing-price",
	EventTypeAuctionEnded:                       "auction-ended",
	EventTypeTimeTrigger:                        "time-trigger",
	EventTypeMarketUpdated:                      "market-updated",
}
