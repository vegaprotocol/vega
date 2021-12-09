package statevar

import (
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
)

// FloatValue is a scalar floating point value.
type FloatValue struct {
	Val float64
}

// Equals returns true of the other value is a scalar floating point with equals value.
func (fv *FloatValue) Equals(other value) bool {
	switch v := other.(type) {
	case *FloatValue:
		return fv.equals(v)
	default:
		return false
	}
}

// equals returns true if the two values are equal.
func (fv *FloatValue) equals(other *FloatValue) bool {
	return fv.withinTolerance(other, num.DecimalZero())
}

// WithinTolerance returns true if the other value is a scalar value and is equal to this scalar value within the given tolerance.
func (fv *FloatValue) WithinTolerance(other value, tolerance num.Decimal) bool {
	switch v := other.(type) {
	case *FloatValue:
		return fv.withinTolerance(v, tolerance)
	default:
		return false
	}
}

// withinTolerance returns true if the two scalar values are equal within the given tolerance
func (fv *FloatValue) withinTolerance(other *FloatValue, tolerance num.Decimal) bool {
	return num.DecimalFromFloat(fv.Val).Sub(num.DecimalFromFloat(other.Val)).Abs().LessThanOrEqual(tolerance)
}

// ToDecimal converts the float scalar to a decimal value
func (fv *FloatValue) ToDecimal() DecimalValue {
	return &DecimalScalarValue{
		Value: num.DecimalFromFloat(fv.Val),
	}
}

//ToProto converts the state variable value to protobuf
func (fv *FloatValue) ToProto() *vega.StateVarValue {
	return &vega.StateVarValue{
		Value: &vega.StateVarValue_ScalarVal{
			ScalarVal: &vega.ScalarValue{
				Value: fv.Val,
			},
		},
	}
}
