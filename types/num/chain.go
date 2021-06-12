package num

import "github.com/shopspring/decimal"

type uChain struct {
	z *Uint
}

// UintChain returns a Uint that supports chainable operations
// The Uint passed to the constructor is the value that will be updated, so be careful
// Things like x := NewUint(0).Add(y, z)
// x.Mul(x, foo)
// can be written as:
// x := UintChain(NewUint(0)).Add(y, z).Mul(foo).Get()
func UintChain(z *Uint) *uChain {
	return &uChain{
		z: z,
	}
}

// Get gets the result of the chained operation (the value of the wrapped uint)
func (c *uChain) Get() *Uint {
	return c.z
}

// Add is equivalent to AddSum
func (c *uChain) Add(vals ...*Uint) *uChain {
	if len(vals) == 0 {
		return c
	}
	c.z.AddSum(vals...)
	return c
}

// Sub subtracts any numbers from the chainable value
func (c *uChain) Sub(vals ...*Uint) *uChain {
	for _, v := range vals {
		c.z.Sub(c.z, v)
	}
	return c
}

// Mul multiplies the current value by x
func (c *uChain) Mul(x *Uint) *uChain {
	c.z.Mul(c.z, x)
	return c
}

// Div divides the current value by x
func (c *uChain) Div(x *Uint) *uChain {
	c.z.Div(c.z, x)
	return c
}

type DecRounding int

const (
	DecFloor DecRounding = iota
	DecRound
	DecCeil
)

type dChain struct {
	d Decimal
}

// UintDecChain returns a chainable decimal from a given uint
// this moves the conversion stuff out from the caller
func UintDecChain(u *Uint) *dChain {
	// @TODO once the updates to the decimal file are merged, call the coversion function from that file
	return &dChain{
		d: decimal.NewFromBigInt(u.u.ToBig(), 0),
	}
}

// DecChain offers the same chainable interface for decimals
func DecChain(d Decimal) *dChain {
	return &dChain{
		d: d,
	}
}

// Get returns the final value
func (d *dChain) Get() Decimal {
	return d.d
}

// GetUint returns the decimal as a uint, returns true on overflow
// pass in type of rounding to apply
// not that the rounding does not affect the underlying decimal value
// rounding is applied to a copy only
func (d *dChain) GetUint(round DecRounding) (*Uint, bool) {
	v := d.d
	switch round {
	case DecFloor:
		v = v.Floor()
	case DecCeil:
		v = v.Ceil()
	case DecRound:
		v = v.Round(0) // we're converting to Uint, so round to 0 places
	}
	return UintFromBig(v.BigInt())
}

// Add adds any number of decimals together
func (d *dChain) Add(vals ...Decimal) *dChain {
	for _, v := range vals {
		d.d = d.d.Add(v)
	}
	return d
}

// Sub subtracts any number of decimals from the chainable value
func (d *dChain) Sub(vals ...Decimal) *dChain {
	for _, v := range vals {
		d.d = d.d.Sub(v)
	}
	return d
}

// Mul multiplies, obviously
func (d *dChain) Mul(x Decimal) *dChain {
	d.d = d.d.Mul(x)
	return d
}

// Div divides
func (d *dChain) Div(x Decimal) *dChain {
	d.d = d.d.Div(x)
	return d
}

// DivRound divides with a specified precision
func (d *dChain) DivRound(x Decimal, precision int32) *dChain {
	d.d = d.d.DivRound(x, precision)
	return d
}
