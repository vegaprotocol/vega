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

package num

import (
	"database/sql/driver"
	"fmt"
	"math/big"

	"github.com/holiman/uint256"
)

var (
	ErrInvalidScanInput = fmt.Errorf("invalid input for Scan")
	intZero             = NewInt(0)
)

// Int a wrapper to a signed big int.
type Int struct {
	// The unsigned version of the integer
	U *Uint
	// The sign of the integer true = positive, false = negative
	s bool
}

func IntFromUint(u *Uint, s bool) *Int {
	return &Int{
		s: s,
		U: u.Clone(),
	}
}

func IntToString(u *Int) string {
	if u != nil {
		return u.String()
	}
	return "0"
}

// IntFromString creates a new Int from a string
// interpreted using the give base.
// A big.Int is used to read the string, so
// all error related to big.Int parsing applied here.
// will return true if an error happened.
func IntFromString(str string, base int) (*Int, bool) {
	b, ok := big.NewInt(0).SetString(str, base)
	if !ok {
		return NewInt(0), true
	}
	return IntFromBig(b)
}

// IntFromBig construct a new Int with a big.Int
// returns true if overflow happened.
func IntFromBig(b *big.Int) (*Int, bool) {
	positive := true
	if b.Sign() < 0 {
		b.Neg(b)
		positive = false
	}

	u, overflow := uint256.FromBig(b)
	if overflow {
		return NewInt(0), true
	}
	return &Int{
		U: &Uint{*u},
		s: positive,
	}, false
}

// IntFromDecimal returns the Int part of a decimal.
func IntFromDecimal(d Decimal) (*Int, bool) {
	dd := d

	// if its negative it'll overflow so need to negate before going to Uint
	if d.IsNegative() {
		dd = d.Neg()
	}
	u, overflow := dd.Uint()
	return &Int{
		U: &Uint{*u},
		s: d.IsPositive(),
	}, overflow
}

// IsNegative tests if the stored value is negative
// true if < 0
// false if >= 0.
func (i *Int) IsNegative() bool {
	return !i.s && !i.U.IsZero()
}

// IsPositive tests if the stored value is positive
// true if > 0
// false if <= 0.
func (i *Int) IsPositive() bool {
	return i.s && !i.U.IsZero()
}

// IsZero tests if the stored value is zero
// true if == 0.
func (i *Int) IsZero() bool {
	return i.U.IsZero()
}

// FlipSign changes the sign of the number from - to + and back again.
func (i *Int) FlipSign() {
	i.s = !i.s
}

// Clone creates a copy of the object so nothing is shared.
func (i Int) Clone() *Int {
	return &Int{
		U: i.U.Clone(),
		s: i.s,
	}
}

func (i Int) EQ(o *Int) bool {
	return i.s == o.s && i.U.EQ(o.U)
}

// GT returns if i > o.
func (i Int) GT(o *Int) bool {
	if i.IsNegative() {
		if o.IsPositive() || o.IsZero() {
			return false
		}

		return i.U.LT(o.U)
	}
	if i.IsPositive() {
		if o.IsZero() || o.IsNegative() {
			return true
		}

		return i.U.GT(o.U)
	}

	return o.IsNegative()
}

func (i Int) GTE(o *Int) bool {
	return i.GT(o) || i.EQ(o)
}

// LT returns if i < o.
func (i Int) LT(o *Int) bool {
	if i.IsNegative() {
		if o.IsPositive() || o.IsZero() {
			return true
		}

		return i.U.GT(o.U)
	}
	if i.IsPositive() {
		if o.IsZero() || o.IsNegative() {
			return false
		}

		return i.U.LT(o.U)
	}

	return o.IsPositive()
}

func (i Int) LTE(o *Int) bool {
	return i.LT(o) || i.EQ(o)
}

func (i Int) Int64() int64 {
	val := int64(i.U.Uint64())
	if i.IsNegative() {
		return -val
	}

	return val
}

// String returns a string version of the number.
func (i Int) String() string {
	val := i.U.String()
	if i.IsNegative() {
		return "-" + val
	}

	return val
}

// Add will add the passed in value to the base value
// i = i + a.
func (i *Int) Add(a *Int) *Int {
	// Handle cases where we have a zero
	if a.IsZero() {
		return i
	}
	if i.IsZero() {
		i.U.Set(a.U)
		i.s = a.s

		return i
	}

	// Handle the easy cases were both are the same sign
	if i.IsPositive() && a.IsPositive() {
		i.U.Add(i.U, a.U)

		return i
	}

	if i.IsNegative() && a.IsNegative() {
		i.U.Add(i.U, a.U)
		return i
	}

	// Now the cases where the signs are different
	if i.IsNegative() {
		if i.U.GTE(a.U) {
			// abs(i) >= a
			i.U.Sub(i.U, a.U)
		} else {
			// abs(i) < a
			i.U.Sub(a.U, i.U)
			i.s = true
		}

		return i
	}
	if i.U.GTE(a.U) {
		// i >= abs(a)
		i.U.Sub(i.U, a.U)
	} else {
		// i < abs(a)
		i.U.Sub(a.U, i.U)
		i.s = false
	}

	return i
}

// Sub will subtract the passed in value from the base value
// i = i - a.
func (i *Int) Sub(a *Int) *Int {
	a.FlipSign()
	i.Add(a)
	a.FlipSign()

	return i
}

// AddSum adds all of the parameters to i
// i = i + a + b + c.
func (i *Int) AddSum(vals ...*Int) *Int {
	for _, x := range vals {
		i.Add(x)
	}

	return i
}

// SubSum subtracts all of the parameters from i
// i = i - a - b - c.
func (i *Int) SubSum(vals ...*Int) *Int {
	for _, x := range vals {
		i.Sub(x)
	}

	return i
}

// Mul will multiply the passed in value to the base value
// i = i * m.
func (i *Int) Mul(m *Int) *Int {
	i.U.Mul(i.U, m.U)
	i.s = i.s == m.s
	return i
}

// Mul will divide the passed in value to the base value
// i = i / m.
func (i *Int) Div(m *Int) *Int {
	i.U.Div(i.U, m.U)
	i.s = i.s == m.s
	return i
}

// Value returns the string representation for SQL queries.
func (i *Int) Value() (driver.Value, error) {
	str := i.String()
	return str, nil
}

// Scan lets Uint.Scan do the heavy lifting, we just check for leading sign characters here.
func (i *Int) Scan(v any) error {
	var str string
	switch vt := v.(type) {
	case string:
		str = vt
	case []byte:
		str = string(vt)
	default:
		return ErrInvalidScanInput
	}
	i.s = true
	// set sign flag, strip leading +/- sign.
	switch str[0:1] {
	case "-":
		i.s = false
		fallthrough
	case "+":
		str = str[1:]
	}
	return i.U.Scan(str)
}

// NewInt creates a new Int with the value of the
// int64 passed as a parameter.
func NewInt(val int64) *Int {
	if val < 0 {
		return &Int{
			U: NewUint(uint64(-val)),
			s: false,
		}
	}

	return &Int{
		U: NewUint(uint64(val)),
		s: true,
	}
}

func IntZero() *Int {
	return intZero.Clone()
}

// NewIntFromUint creates a new Int with the value of the
// uint passed as a parameter.
func NewIntFromUint(val *Uint) *Int {
	return &Int{
		U: val,
		s: true,
	}
}
