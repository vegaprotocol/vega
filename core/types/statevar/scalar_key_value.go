// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
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
	"code.vegaprotocol.io/vega/libs/num"
)

// DecimalScalar is a scalar floating point value.
type DecimalScalar struct {
	Val num.Decimal
}

// Equals returns true of the other value is a scalar floating point with equals value.
func (dv *DecimalScalar) Equals(other value) bool {
	switch v := other.(type) {
	case *DecimalScalar:
		return dv.equals(v)
	default:
		return false
	}
}

// equals returns true if the two values are equal.
func (dv *DecimalScalar) equals(other *DecimalScalar) bool {
	return dv.withinTolerance(other, num.DecimalZero())
}

// WithinTolerance returns true if the other value is a scalar value and is equal to this scalar value within the given tolerance.
func (dv *DecimalScalar) WithinTolerance(other value, tolerance num.Decimal) bool {
	switch v := other.(type) {
	case *DecimalScalar:
		return dv.withinTolerance(v, tolerance)
	default:
		return false
	}
}

// withinTolerance returns true if the two scalar values are equal within the given tolerance.
func (dv *DecimalScalar) withinTolerance(other *DecimalScalar, tolerance num.Decimal) bool {
	return dv.Val.Sub(other.Val).Abs().LessThanOrEqual(tolerance)
}

// ToProto converts the state variable value to protobuf.
func (fv *DecimalScalar) ToProto() *vega.StateVarValue {
	return &vega.StateVarValue{
		Value: &vega.StateVarValue_ScalarVal{
			ScalarVal: &vega.ScalarValue{
				Value: fv.Val.String(),
			},
		},
	}
}
