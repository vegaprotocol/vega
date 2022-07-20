// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package statevar

import (
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
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
