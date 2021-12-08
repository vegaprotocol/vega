package statevar

import (
	"math"

	"code.vegaprotocol.io/vega/types/num"
)

type FloatMatrix struct {
	Val [][]float64
}

// Equals returns true of the other value is a matrix of floating point values with the same shape and equals values.
func (fv *FloatMatrix) Equals(other value) bool {
	switch v := other.(type) {
	case *FloatMatrix:
		return fv.equals(v)
	default:
		return false
	}
}

// equals returns true if the two matrices are equal.
func (fv *FloatMatrix) equals(other *FloatMatrix) bool {
	return fv.withinTolerance(other, 0)
}

// WithinTolerance returns true if the other value is a matrix and has the same shape and values in the same index are within the given tolerance of each other.
func (fv *FloatMatrix) WithinTolerance(other value, tolerance float64) bool {
	switch v := other.(type) {
	case *FloatMatrix:
		return fv.withinTolerance(v, tolerance)
	default:
		return false
	}
}

// withinTolerance retunrs true if the two matrices have the same shape and values in the same index are within tolerance of each other
func (fv *FloatMatrix) withinTolerance(other *FloatMatrix, tolerance float64) bool {
	if len(fv.Val) != len(other.Val) {
		return false
	}
	for i := range fv.Val {
		if len(fv.Val[i]) != len(other.Val[i]) {
			return false
		}
		for j := range fv.Val[i] {
			if math.Abs(fv.Val[i][j]-other.Val[i][j]) > tolerance {
				return false
			}
		}
	}
	return true
}

// ToDecimal converts the float matrix to decimal matrix
func (fv *FloatMatrix) ToDecimal() DecimalValue {
	rows := make([][]num.Decimal, 0, len(fv.Val))
	for _, r := range fv.Val {
		cols := make([]num.Decimal, 0, len(r))
		for _, c := range r {
			cols = append(cols, num.DecimalFromFloat(c))
		}
		rows = append(rows, cols)
	}

	return &DecimalMatrixValue{
		Value: rows,
	}
}
