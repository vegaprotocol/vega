package num

type WrappedDecimal struct {
	u *Uint
	d Decimal
}

// NewWrappedDecimal returns a new instance of a decimal coupled with the Uint representation that has been chosen for it
func NewWrappedDecimal(integerRepresentation *Uint, underlyingValue Decimal) WrappedDecimal {
	return WrappedDecimal{u: integerRepresentation, d: underlyingValue}
}

// Representation returns the Uint representation of a decimal that has been affixed when calling the constructor
func (w WrappedDecimal) Representation() *Uint {
	return w.u.Clone()
}

// Original returns the underlying decimal value
func (w WrappedDecimal) Original() Decimal {
	return w.d
}
