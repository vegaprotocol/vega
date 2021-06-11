package num

import (
	"fmt"
	"math/big"

	"github.com/holiman/uint256"
)

// Uint A wrapper for a big unsigned int
type Uint struct {
	u uint256.Int
}

// NewUint creates a new Uint with the value of the
// uint64 passed as a paramter.
func NewUint(val uint64) *Uint {
	return &Uint{*uint256.NewInt(val)}
}

// Min returns the smallest of the 2 numbers
func Min(a, b *Uint) *Uint {
	if a.LT(b) {
		return a
	}
	return b
}

// Max returns the largest of the 2 numbers
func Max(a, b *Uint) *Uint {
	if a.GT(b) {
		return a
	}
	return b
}

// FromBig construct a new Uint with a big.Int
// returns true if overflow happened
func UintFromBig(b *big.Int) (*Uint, bool) {
	u, ok := uint256.FromBig(b)
	// ok means an overflow happened
	if ok {
		return NewUint(0), true
	}
	return &Uint{*u}, false
}

func UintFromDecimal(d Decimal) (*Uint, bool) {
	return UintFromBig(d.BigInt())
}

func (u *Uint) ToDecimal() Decimal {
	return DecimalFromUint(u)
}

// FromString created a new Uint from a string
// interpreted using the give base.
// A big.Int is used to read the string, so
// all error related to big.Int parsing applied here.
// will return true if an error/overflow happened
func UintFromString(str string, base int) (*Uint, bool) {
	b, ok := big.NewInt(0).SetString(str, base)
	if !ok {
		return NewUint(0), true
	}
	return UintFromBig(b)
}

// Sum just removes the need to write num.NewUint(0).Sum(x, y, z)
// so you can write num.Sum(x, y, z) instead, equivalent to x + y + z
func Sum(vals ...*Uint) *Uint {
	return NewUint(0).AddSum(vals...)
}

func (z *Uint) Set(oth *Uint) *Uint {
	z.u.Set(&oth.u)
	return z
}

func (z *Uint) SetUint64(val uint64) *Uint {
	z.u.SetUint64(val)
	return z
}

func (z Uint) Uint64() uint64 {
	return z.u.Uint64()
}

func (z Uint) BigInt() *big.Int {
	return z.u.ToBig()
}

func (z Uint) Float64() float64 {
	d := DecimalFromUint(&z)
	retVal, _ := d.Float64()
	return retVal
}

// Add will add x and y then store the result
// into u
// this is equivalent to:
// `z = x + y`
// u is returned for convenience, no
// new variable is created.
func (z *Uint) Add(x, y *Uint) *Uint {
	z.u.Add(&x.u, &y.u)
	return z
}

// AddSum adds multiple values at the same time to a given uint
// so x.AddSum(y, z) is equivalent to x + y + z
func (z *Uint) AddSum(vals ...*Uint) *Uint {
	for _, x := range vals {
		z.u.Add(&z.u, &x.u)
	}
	return z
}

// AddOverflow will substract y to x then store the result
// into u
// this is equivalent to:
// `z = x - y`
// u is returned for convenience, no
// new variable is created.
// False is returned if an overflow occurred
func (z *Uint) AddOverflow(x, y *Uint) (*Uint, bool) {
	_, ok := z.u.AddOverflow(&x.u, &y.u)
	return z, ok
}

// Sub will substract y from x then store the result
// into u
// this is equivalent to:
// `u = x - y`
// u is returned for convenience, no
// new variable is created.
func (u *Uint) Sub(x, y *Uint) *Uint {
	u.u.Sub(&x.u, &y.u)
	return u
}

// SubOverflow will substract y to x then store the result
// into u
// this is equivalent to:
// `z = x - y`
// u is returned for convenience, no
// new variable is created.
// False is returned if an overflow occured
func (z *Uint) SubOverflow(x, y *Uint) (*Uint, bool) {
	_, ok := z.u.SubOverflow(&x.u, &y.u)
	return z, ok
}

// Delta will subtract y from x and store the result
// unless x-y overflowed, in which case the neg field will be set
// and the result of y - x is set instead
func (z *Uint) Delta(x, y *Uint) (*Uint, bool) {
	// y is the bigger value - swap the two
	if y.GT(x) {
		_ = z.Sub(y, x)
		return z, true
	}
	_ = z.Sub(x, y)
	return z, false
}

// Mul will multiply x and y then store the result
// into z
// this is equivalent to:
// `z = x * y`
// z is returned for convenience, no
// new variable is created.
func (z *Uint) Mul(x, y *Uint) *Uint {
	z.u.Mul(&x.u, &y.u)
	return z
}

// Div will divide x by y then store the result
// into u
// this is equivalent to:
// `u = x / y`
// u is returned for convenience, no
// new variable is created.
func (u *Uint) Div(x, y *Uint) *Uint {
	u.u.Div(&x.u, &y.u)
	return u
}

// LT with check if the value stored in u is
// lesser than oth
// this is equivalent to:
// `u < oth`
func (u Uint) LT(oth *Uint) bool {
	return u.u.Lt(&oth.u)
}

// LTUint64 with check if the value stored in u is
// lesser than oth
// this is equivalent to:
// `u < oth`
func (u Uint) LTUint64(oth uint64) bool {
	return u.u.LtUint64(oth)
}

// LTE with check if the value stored in u is
// lesser than or equal to oth
// this is equivalent to:
// `u <= oth`
func (u Uint) LTE(oth *Uint) bool {
	return u.u.Lt(&oth.u) || u.u.Eq(&oth.u)
}

// LTEUint64 with check if the value stored in u is
// lesser than or equal to oth
// this is equivalent to:
// `u <= oth`
func (u Uint) LTEUint64(oth uint64) bool {
	return u.u.LtUint64(oth) || u.EQUint64(oth)
}

// EQ with check if the value stored in u is
// equal to oth
// this is equivalent to:
// `u == oth`
func (u Uint) EQ(oth *Uint) bool {
	return u.u.Eq(&oth.u)
}

// EQUint64 with check if the value stored in u is
// equal to oth
// this is equivalent to:
// `u == oth`
func (u Uint) EQUint64(oth uint64) bool {
	return u.u.Eq(uint256.NewInt(oth))
}

// NEQ with check if the value stored in u is
// different than oth
// this is equivalent to:
// `u != oth`
func (u Uint) NEQ(oth *Uint) bool {
	return !u.u.Eq(&oth.u)
}

// NEQUint64 with check if the value stored in u is
// different than oth
// this is equivalent to:
// `u != oth`
func (u Uint) NEQUint64(oth uint64) bool {
	return !u.u.Eq(uint256.NewInt(oth))
}

// GT with check if the value stored in u is
// greater than oth
// this is equivalent to:
// `u > oth`
func (u Uint) GT(oth *Uint) bool {
	return u.u.Gt(&oth.u)
}

// GTUint64 with check if the value stored in u is
// greater than oth
// this is equivalent to:
// `u > oth`
func (u Uint) GTUint64(oth uint64) bool {
	return u.u.GtUint64(oth)
}

// GTE with check if the value stored in u is
// greath than or equal to oth
// this is equivalent to:
// `u >= oth`
func (u Uint) GTE(oth *Uint) bool {
	return u.u.Gt(&oth.u) || u.u.Eq(&oth.u)
}

// GTEUint64 with check if the value stored in u is
// greath than or equal to oth
// this is equivalent to:
// `u >= oth`
func (u Uint) GTEUint64(oth uint64) bool {
	return u.u.GtUint64(oth) || u.EQUint64(oth)
}

// IsZero return wether u == 0 or not
func (u Uint) IsZero() bool {
	return u.u.IsZero()
}

// Copy create a copy of the uint
// this if the equivalenht to:
// z = x
func (z *Uint) Copy(x *Uint) *Uint {
	z.u = x.u
	return z
}

// Clone create copy of this value
// this is the equivalent to:
// x := z
func (z Uint) Clone() *Uint {
	return &Uint{z.u}
}

// Hex returns the hexadecimal representation
// of the stored value
func (u Uint) Hex() string {
	return u.u.Hex()
}

// String returns the stored value as a string
// this is internally using big.Int.String()
func (u Uint) String() string {
	return u.u.ToBig().String()
}

// Format implement fmt.Formatter
func (u Uint) Format(s fmt.State, ch rune) {
	u.u.Format(s, ch)
}

// Bytes return the internal representation
// of the Uint as [32]bytes, BigEndian encoded
// array
func (u Uint) Bytes() [32]byte {
	return u.u.Bytes32()
}
