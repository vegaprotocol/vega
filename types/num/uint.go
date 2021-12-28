package num

import (
	"fmt"
	"math/big"

	"github.com/holiman/uint256"
)

var (
	// max uint256 value.
	big1    = big.NewInt(1)
	maxU256 = new(big.Int).Sub(new(big.Int).Lsh(big1, 256), big1)

	// initialise max variable.
	maxUint = setMaxUint()
	zero    = NewUint(0)
)

// Uint A wrapper for a big unsigned int.
type Uint struct {
	u uint256.Int
}

// NewUint creates a new Uint with the value of the
// uint64 passed as a parameter.
func NewUint(val uint64) *Uint {
	return &Uint{*uint256.NewInt(val)}
}

func Zero() *Uint {
	return zero.Clone()
}

// only called once, to initialise maxUint.
func setMaxUint() *Uint {
	b, _ := UintFromBig(maxU256)
	return b
}

// MaxUint returns max value for uint256.
func MaxUint() *Uint {
	return maxUint.Clone()
}

// Min returns the smallest of the 2 numbers.
func Min(a, b *Uint) *Uint {
	if a.LT(b) {
		return a.Clone()
	}
	return b.Clone()
}

// Max returns the largest of the 2 numbers.
func Max(a, b *Uint) *Uint {
	if a.GT(b) {
		return a.Clone()
	}
	return b.Clone()
}

// UintFromHex instantiate a uint from and hex string.
func UintFromHex(hex string) (*Uint, error) {
	u, err := uint256.FromHex(hex)
	if err != nil {
		return nil, err
	}
	return &Uint{*u}, nil
}

// UintFromBig construct a new Uint with a big.Int
// returns true if overflow happened.
func UintFromBig(b *big.Int) (*Uint, bool) {
	u, ok := uint256.FromBig(b)
	// ok means an overflow happened
	if ok {
		return NewUint(0), true
	}
	return &Uint{*u}, false
}

// UintFromBytes allows for the conversion from Uint.Bytes() back to a Uint.
func UintFromBytes(b []byte) *Uint {
	u := &Uint{
		u: uint256.Int{},
	}
	u.u.SetBytes(b)
	return u
}

// UintFromDecimal returns a decimal version of the Uint, setting the bool to true if overflow occurred.
func UintFromDecimal(d Decimal) (*Uint, bool) {
	u, ok := d.Uint()
	return &Uint{*u}, ok
}

// ToDecimal returns the value of the Uint as a Decimal.
func (u *Uint) ToDecimal() Decimal {
	return DecimalFromUint(u)
}

// UintFromString created a new Uint from a string
// interpreted using the give base.
// A big.Int is used to read the string, so
// all error related to big.Int parsing applied here.
// will return true if an error/overflow happened.
func UintFromString(str string, base int) (*Uint, bool) {
	b, ok := big.NewInt(0).SetString(str, base)
	if !ok {
		return NewUint(0), true
	}
	return UintFromBig(b)
}

// Sum just removes the need to write num.NewUint(0).Sum(x, y, z)
// so you can write num.Sum(x, y, z) instead, equivalent to x + y + z.
func Sum(vals ...*Uint) *Uint {
	return NewUint(0).AddSum(vals...)
}

func (u *Uint) Set(oth *Uint) *Uint {
	u.u.Set(&oth.u)
	return u
}

func (u *Uint) SetUint64(val uint64) *Uint {
	u.u.SetUint64(val)
	return u
}

func (u Uint) Uint64() uint64 {
	return u.u.Uint64()
}

func (u Uint) BigInt() *big.Int {
	return u.u.ToBig()
}

func (u Uint) Float64() float64 {
	d := DecimalFromUint(&u)
	retVal, _ := d.Float64()
	return retVal
}

// Add will add x and y then store the result
// into u
// this is equivalent to:
// `u = x + y`
// u is returned for convenience, no
// new variable is created.
func (u *Uint) Add(x, y *Uint) *Uint {
	u.u.Add(&x.u, &y.u)
	return u
}

// AddSum adds multiple values at the same time to a given uint
// so x.AddSum(y, z) is equivalent to x + y + z.
func (u *Uint) AddSum(vals ...*Uint) *Uint {
	for _, x := range vals {
		u.u.Add(&u.u, &x.u)
	}
	return u
}

// AddOverflow will subtract y to x then store the result
// into u
// this is equivalent to:
// `u = x - y`
// u is returned for convenience, no
// new variable is created.
// False is returned if an overflow occurred.
func (u *Uint) AddOverflow(x, y *Uint) (*Uint, bool) {
	_, ok := u.u.AddOverflow(&x.u, &y.u)
	return u, ok
}

// Sub will subtract y from x then store the result
// into u
// this is equivalent to:
// `u = x - y`
// u is returned for convenience, no
// new variable is created.
func (u *Uint) Sub(x, y *Uint) *Uint {
	u.u.Sub(&x.u, &y.u)
	return u
}

// SubOverflow will subtract y to x then store the result
// into u
// this is equivalent to:
// `u = x - y`
// u is returned for convenience, no
// new variable is created.
// False is returned if an overflow occurred.
func (u *Uint) SubOverflow(x, y *Uint) (*Uint, bool) {
	_, ok := u.u.SubOverflow(&x.u, &y.u)
	return u, ok
}

// Delta will subtract y from x and store the result
// unless x-y overflowed, in which case the neg field will be set
// and the result of y - x is set instead.
func (u *Uint) Delta(x, y *Uint) (*Uint, bool) {
	// y is the bigger value - swap the two
	if y.GT(x) {
		_ = u.Sub(y, x)
		return u, true
	}
	_ = u.Sub(x, y)
	return u, false
}

// DeltaI will subtract y from x and store the result.
func (u *Uint) DeltaI(x, y *Uint) *Int {
	d, s := u.Delta(x, y)
	return IntFromUint(d, !s)
}

// Mul will multiply x and y then store the result
// into u
// this is equivalent to:
// `u = x * y`
// u is returned for convenience, no
// new variable is created.
func (u *Uint) Mul(x, y *Uint) *Uint {
	u.u.Mul(&x.u, &y.u)
	return u
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

// Mod sets u to the modulus x%y for y != 0 and returns u.
// If y == 0, u is set to 0.
func (u *Uint) Mod(x, y *Uint) *Uint {
	u.u.Mod(&x.u, &y.u)
	return u
}

func (u *Uint) Exp(x, y *Uint) *Uint {
	u.u.Exp(&x.u, &y.u)
	return u
}

// LT with check if the value stored in u is
// lesser than oth
// this is equivalent to:
// `u < oth`.
func (u Uint) LT(oth *Uint) bool {
	return u.u.Lt(&oth.u)
}

// LTUint64 with check if the value stored in u is
// lesser than oth
// this is equivalent to:
// `u < oth`.
func (u Uint) LTUint64(oth uint64) bool {
	return u.u.LtUint64(oth)
}

// LTE with check if the value stored in u is
// lesser than or equal to oth
// this is equivalent to:
// `u <= oth`.
func (u Uint) LTE(oth *Uint) bool {
	return u.u.Lt(&oth.u) || u.u.Eq(&oth.u)
}

// LTEUint64 with check if the value stored in u is
// lesser than or equal to oth
// this is equivalent to:
// `u <= oth`.
func (u Uint) LTEUint64(oth uint64) bool {
	return u.u.LtUint64(oth) || u.EQUint64(oth)
}

// EQ with check if the value stored in u is
// equal to oth
// this is equivalent to:
// `u == oth`.
func (u Uint) EQ(oth *Uint) bool {
	return u.u.Eq(&oth.u)
}

// EQUint64 with check if the value stored in u is
// equal to oth
// this is equivalent to:
// `u == oth`.
func (u Uint) EQUint64(oth uint64) bool {
	return u.u.Eq(uint256.NewInt(oth))
}

// NEQ with check if the value stored in u is
// different than oth
// this is equivalent to:
// `u != oth`.
func (u Uint) NEQ(oth *Uint) bool {
	return !u.u.Eq(&oth.u)
}

// NEQUint64 with check if the value stored in u is
// different than oth
// this is equivalent to:
// `u != oth`.
func (u Uint) NEQUint64(oth uint64) bool {
	return !u.u.Eq(uint256.NewInt(oth))
}

// GT with check if the value stored in u is
// greater than oth
// this is equivalent to:
// `u > oth`.
func (u Uint) GT(oth *Uint) bool {
	return u.u.Gt(&oth.u)
}

// GTUint64 with check if the value stored in u is
// greater than oth
// this is equivalent to:
// `u > oth`.
func (u Uint) GTUint64(oth uint64) bool {
	return u.u.GtUint64(oth)
}

// GTE with check if the value stored in u is
// greater than or equal to oth
// this is equivalent to:
// `u >= oth`.
func (u Uint) GTE(oth *Uint) bool {
	return u.u.Gt(&oth.u) || u.u.Eq(&oth.u)
}

// GTEUint64 with check if the value stored in u is
// greater than or equal to oth
// this is equivalent to:
// `u >= oth`.
func (u Uint) GTEUint64(oth uint64) bool {
	return u.u.GtUint64(oth) || u.EQUint64(oth)
}

// IsZero return whether u == 0 or not.
func (u Uint) IsZero() bool {
	return u.u.IsZero()
}

// IsNegative returns whether the value is < 0.
func (u Uint) IsNegative() bool {
	return u.u.Sign() == -1
}

// Copy create a copy of the uint
// this if the equivalent to:
// u = x.
func (u *Uint) Copy(x *Uint) *Uint {
	u.u = x.u
	return u
}

// Clone create copy of this value
// this is the equivalent to:
// x := u.
func (u Uint) Clone() *Uint {
	return &Uint{u.u}
}

// Hex returns the hexadecimal representation
// of the stored value.
func (u Uint) Hex() string {
	return u.u.Hex()
}

// String returns the stored value as a string
// this is internally using big.Int.String().
func (u Uint) String() string {
	return u.u.ToBig().String()
}

// Format implement fmt.Formatter.
func (u Uint) Format(s fmt.State, ch rune) {
	u.u.Format(s, ch)
}

// Bytes return the internal representation
// of the Uint as [32]bytes, BigEndian encoded
// array.
func (u Uint) Bytes() [32]byte {
	return u.u.Bytes32()
}

// UintToUint64 convert a uint to uint64
// return 0 if nil.
func UintToUint64(u *Uint) uint64 {
	if u != nil {
		return u.Uint64()
	}
	return 0
}

// UintToString convert a uint to uint64
// return "0" if nil.
func UintToString(u *Uint) string {
	if u != nil {
		return u.String()
	}
	return "0"
}
