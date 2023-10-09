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
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
)

type DecimalVector struct {
	Val []num.Decimal
}

// Equals returns true of the other value is a vector of floating point values with the same shape and equals values.
func (dv *DecimalVector) Equals(other value) bool {
	switch v := other.(type) {
	case *DecimalVector:
		return dv.equals(v)
	default:
		return false
	}
}

// equals returns true if the two vectors are equal.
func (dv *DecimalVector) equals(other *DecimalVector) bool {
	return dv.withinTolerance(other, num.DecimalZero())
}

// WithinTolerance returns true if the other value is a vector and has the same shape and values in the same index are within the given tolerance of each other.
func (dv *DecimalVector) WithinTolerance(other value, tolerance num.Decimal) bool {
	switch v := other.(type) {
	case *DecimalVector:
		return dv.withinTolerance(v, tolerance)
	default:
		return false
	}
}

// withinTolerance returns true if the two vectors have the same shape and values in the same index are within the given tolerance of each other.
func (dv *DecimalVector) withinTolerance(other *DecimalVector, tolerance num.Decimal) bool {
	if len(dv.Val) != len(other.Val) {
		return false
	}
	for i := range dv.Val {
		// we probably don't need the tolerance on the tolerance check but for testing its useful
		if dv.Val[i].Sub(other.Val[i]).Abs().GreaterThan(tolerance) {
			return false
		}
	}

	return true
}

// ToProto converts the state variable value to protobuf.
func (dv *DecimalVector) ToProto() *vega.StateVarValue {
	values := make([]string, 0, len(dv.Val))
	for _, v := range dv.Val {
		values = append(values, v.String())
	}

	return &vega.StateVarValue{
		Value: &vega.StateVarValue_VectorVal{
			VectorVal: &vega.VectorValue{
				Value: values,
			},
		},
	}
}
