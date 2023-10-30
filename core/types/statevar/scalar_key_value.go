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
func (dv *DecimalScalar) ToProto() *vega.StateVarValue {
	return &vega.StateVarValue{
		Value: &vega.StateVarValue_ScalarVal{
			ScalarVal: &vega.ScalarValue{
				Value: dv.Val.String(),
			},
		},
	}
}
