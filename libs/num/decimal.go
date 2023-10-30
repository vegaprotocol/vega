// Copyright (C) 2023  Gobalsky Labs Limited
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
	"errors"
	"math/big"

	"github.com/shopspring/decimal"
)

type Decimal = decimal.Decimal

var (
	dzero      = decimal.Zero
	d1         = decimal.NewFromFloat(1)
	dm1        = decimal.NewFromFloat(-1)
	d2         = decimal.NewFromFloat(2)
	maxDecimal = decimal.NewFromBigInt(maxU256, 0)
	e          = MustDecimalFromString("2.7182818285")
)

func MustDecimalFromString(f string) Decimal {
	d, err := DecimalFromString(f)
	if err != nil {
		panic(err)
	}
	return d
}

func DecimalOne() Decimal {
	return d1
}

func DecimalMinusOne() Decimal {
	return dm1
}

func DecimalTwo() Decimal {
	return d2
}

func DecimalZero() Decimal {
	return dzero
}

func DecimalE() Decimal {
	return e
}

func MaxDecimal() Decimal {
	return maxDecimal
}

func NewDecimalFromFloat(f float64) Decimal {
	return decimal.NewFromFloat(f)
}

func NewDecimalFromBigInt(value *big.Int, exp int32) Decimal {
	return decimal.NewFromBigInt(value, exp)
}

func DecimalFromUint(u *Uint) Decimal {
	return decimal.NewFromUint(&u.u)
}

func DecimalFromInt(u *Int) Decimal {
	d := decimal.NewFromUint(&u.U.u)
	if u.IsNegative() {
		return d.Neg()
	}
	return d
}

func DecimalFromInt64(i int64) Decimal {
	return decimal.NewFromInt(i)
}

func DecimalFromFloat(v float64) Decimal {
	return decimal.NewFromFloat(v)
}

func DecimalFromString(s string) (Decimal, error) {
	return decimal.NewFromString(s)
}

func DecimalPart(a Decimal) Decimal {
	return a.Sub(a.Floor())
}

func MaxD(a, b Decimal) Decimal {
	if a.GreaterThan(b) {
		return a
	}
	return b
}

func MinD(a, b Decimal) Decimal {
	if a.LessThan(b) {
		return a
	}
	return b
}

// calculates the mean of a given slice.
func Mean(numbers []Decimal) (Decimal, error) {
	if len(numbers) == 0 {
		return DecimalZero(), errors.New("cannot calculate the mean of an empty list")
	}
	total := DecimalZero()
	for _, num := range numbers {
		total = total.Add(num)
	}
	return total.Div(DecimalFromInt64(int64(len(numbers)))), nil
}

// calculates the variance of a decimal slice.
func Variance(numbers []Decimal) (Decimal, error) {
	if len(numbers) == 0 {
		return DecimalZero(), errors.New("cannot calculate the variance of an empty list")
	}
	m, _ := Mean(numbers)
	total := DecimalZero()
	for _, num := range numbers {
		total = total.Add(num.Sub(m).Pow(d2))
	}
	return total.Div(DecimalFromInt64(int64(len(numbers)))), nil
}

func UnmarshalBinaryDecimal(data []byte) (Decimal, error) {
	d := decimal.New(0, 1)
	err := d.UnmarshalBinary(data)
	return d, err
}
