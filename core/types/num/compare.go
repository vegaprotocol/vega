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

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64
}

type Unsigned interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

type Num interface {
	Signed | Unsigned
}

// MaxV generic max of any numeric values.
func MaxV[T Num](a, b T) T {
	if a > b {
		return a
	}
	return b
}

// MinV generic min of numneric values.
func MinV[T Num](a, b T) T {
	if a > b {
		return b
	}
	return a
}

// AbsV generic absolute value function of signed primitives.
func AbsV[T Signed](a T) T {
	var b T // get the nil value
	if a < b {
		return -a
	}
	return a
}
