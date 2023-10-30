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

type DecimalMatrix struct {
	Val [][]num.Decimal
}

// Equals returns true of the other value is a matrix of floating point values with the same shape and equals values.
func (dm *DecimalMatrix) Equals(other value) bool {
	switch v := other.(type) {
	case *DecimalMatrix:
		return dm.equals(v)
	default:
		return false
	}
}

// equals returns true if the two matrices are equal.
func (dm *DecimalMatrix) equals(other *DecimalMatrix) bool {
	return dm.withinTolerance(other, num.DecimalZero())
}

// WithinTolerance returns true if the other value is a matrix and has the same shape and values in the same index are within the given tolerance of each other.
func (dm *DecimalMatrix) WithinTolerance(other value, tolerance num.Decimal) bool {
	switch v := other.(type) {
	case *DecimalMatrix:
		return dm.withinTolerance(v, tolerance)
	default:
		return false
	}
}

// withinTolerance retunrs true if the two matrices have the same shape and values in the same index are within tolerance of each other.
func (dm *DecimalMatrix) withinTolerance(other *DecimalMatrix, tolerance num.Decimal) bool {
	if len(dm.Val) != len(other.Val) {
		return false
	}
	for i := range dm.Val {
		if len(dm.Val[i]) != len(other.Val[i]) {
			return false
		}
		for j := range dm.Val[i] {
			if dm.Val[i][j].Sub(other.Val[i][j]).Abs().GreaterThan(tolerance) {
				return false
			}
		}
	}
	return true
}

// ToProto converts the state variable value to protobuf.
func (dm DecimalMatrix) ToProto() *vega.StateVarValue {
	rows := make([]*vega.VectorValue, 0, len(dm.Val))
	for _, fvi := range dm.Val {
		fviAsString := make([]string, 0, len(fvi))
		for _, dv := range fvi {
			fviAsString = append(fviAsString, dv.String())
		}
		rows = append(rows, &vega.VectorValue{
			Value: fviAsString,
		})
	}
	return &vega.StateVarValue{
		Value: &vega.StateVarValue_MatrixVal{
			MatrixVal: &vega.MatrixValue{
				Value: rows,
			},
		},
	}
}
