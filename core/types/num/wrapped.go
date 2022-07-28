// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package num

type WrappedDecimal struct {
	u *Uint
	d Decimal
}

// NewWrappedDecimal returns a new instance of a decimal coupled with the Uint representation that has been chosen for it.
func NewWrappedDecimal(integerRepresentation *Uint, underlyingValue Decimal) WrappedDecimal {
	return WrappedDecimal{u: integerRepresentation, d: underlyingValue}
}

// Representation returns the Uint representation of a decimal that has been affixed when calling the constructor.
func (w WrappedDecimal) Representation() *Uint {
	return w.u.Clone()
}

// Original returns the underlying decimal value.
func (w WrappedDecimal) Original() Decimal {
	return w.d
}
