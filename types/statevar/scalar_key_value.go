package statevar

import (
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
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

// ToDecimal converts the float scalar to a decimal value.
func (dv *DecimalScalar) ToDecimal() DecimalValue {
	return dv
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
