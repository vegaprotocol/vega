package statevar

import (
	"math"

	"code.vegaprotocol.io/vega/types/num"
)

type FloatVector struct {
	Val []float64
}

// Equals returns true of the other value is a vector of floating point values with the same shape and equals values.
func (fv *FloatVector) Equals(other value) bool {
	switch v := other.(type) {
	case *FloatVector:
		return fv.equals(v)
	default:
		return false
	}
}

// equals returns true if the two vectors are equal.
func (fv *FloatVector) equals(other *FloatVector) bool {
	return fv.withinTolerance(other, 0)
}

// WithinTolerance returns true if the other value is a vector and has the same shape and values in the same index are within the given tolerance of each other
func (fv *FloatVector) WithinTolerance(other value, tolerance float64) bool {
	switch v := other.(type) {
	case *FloatVector:
		return fv.withinTolerance(v, tolerance)
	default:
		return false
	}
}

// withinTolerance returns true if the two vectors have the same shape and values in the same index are within the given tolerance of each other
func (fv *FloatVector) withinTolerance(other *FloatVector, tolerance float64) bool {
	if len(fv.Val) != len(other.Val) {
		return false
	}
	for i := range fv.Val {
		// we probably don't need the tolerance on the tolerance check but for testing its useful
		if math.Abs(fv.Val[i]-other.Val[i]) > tolerance {
			return false
		}
	}

	return true
}

// ToDecimal converts the float vector to a vector of decimals
func (fv *FloatVector) ToDecimal() DecimalValue {
	vec := make([]num.Decimal, 0, len(fv.Val))
	for _, v := range fv.Val {
		vec = append(vec, num.DecimalFromFloat(v))
	}
	return &DecimalVectorValue{
		Value: vec,
	}
}
