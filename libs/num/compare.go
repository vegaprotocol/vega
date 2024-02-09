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
