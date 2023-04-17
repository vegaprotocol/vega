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

// MaxAbs - get max value based on absolute values of abolute vals.
func MaxAbs[T Signed](vals ...T) T {
	var r, m T
	for _, v := range vals {
		av := v
		if av < 0 {
			av *= -1
		}
		if av > m {
			r = v
			m = av // current max abs is av
		}
	}
	return r
}

// CmpV compares 2 numeric values of any type, we attempt to cast T1 to T2 and back to see if that conversion
// loses any information, if  no data is lost this way, we compare both values as T2, otherwise we compare both as T1.
func CmpV[T1 Num, T2 Num](a T1, b T2) bool {
	if a2 := T2(a); a == T1(a2) {
		// cast to T2 can be cast back to T1 -> no information is lost, and so T2(a) can be safely compared to b
		return a2 == b
	}
	// information was lost in T2(a), either a is float and b is int, or b is float32 vs a float64
	// either way T1(b) is safe
	return a == T1(b)
}
