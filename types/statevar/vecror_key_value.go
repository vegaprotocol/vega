package statevar

import (
	"code.vegaprotocol.io/protos/vega"
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
	return fv.withinTolerance(other, num.DecimalZero())
}

// WithinTolerance returns true if the other value is a vector and has the same shape and values in the same index are within the given tolerance of each other.
func (fv *FloatVector) WithinTolerance(other value, tolerance num.Decimal) bool {
	switch v := other.(type) {
	case *FloatVector:
		return fv.withinTolerance(v, tolerance)
	default:
		return false
	}
}

// withinTolerance returns true if the two vectors have the same shape and values in the same index are within the given tolerance of each other.
func (fv *FloatVector) withinTolerance(other *FloatVector, tolerance num.Decimal) bool {
	if len(fv.Val) != len(other.Val) {
		return false
	}
	for i := range fv.Val {
		// we probably don't need the tolerance on the tolerance check but for testing its useful
		if num.DecimalFromFloat(fv.Val[i]).Sub(num.DecimalFromFloat(other.Val[i])).Abs().GreaterThan(tolerance) {
			return false
		}
	}

	return true
}

// ToDecimal converts the float vector to a vector of decimals.
func (fv *FloatVector) ToDecimal() DecimalValue {
	vec := make([]num.Decimal, 0, len(fv.Val))
	for _, v := range fv.Val {
		vec = append(vec, num.DecimalFromFloat(v))
	}
	return &DecimalVectorValue{
		Value: vec,
	}
}

//ToProto converts the state variable value to protobuf.
func (fv *FloatVector) ToProto() *vega.StateVarValue {
	return &vega.StateVarValue{
		Value: &vega.StateVarValue_VectorVal{
			VectorVal: &vega.VectorValue{
				Value: fv.Val,
			},
		},
	}
}
