// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package num

import (
	"math/big"

	"github.com/shopspring/decimal"
)

type Decimal = decimal.Decimal

var (
	dzero      = decimal.Zero
	d1         = decimal.NewFromFloat(1)
	maxDecimal = decimal.NewFromBigInt(maxU256, 0)
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

func DecimalZero() Decimal {
	return dzero
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
