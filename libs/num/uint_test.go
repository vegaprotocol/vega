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

package num_test

import (
	"fmt"
	"math/big"
	"testing"

	"code.vegaprotocol.io/vega/libs/num"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUint256Constructors(t *testing.T) {
	var expected uint64 = 42

	t.Run("test from uint64", func(t *testing.T) {
		n := num.NewUint(expected)
		assert.Equal(t, expected, n.Uint64())
	})

	t.Run("test from string", func(t *testing.T) {
		n, overflow := num.UintFromString("42", 10)
		assert.False(t, overflow)
		assert.Equal(t, expected, n.Uint64())
	})

	t.Run("test from big", func(t *testing.T) {
		n, overflow := num.UintFromBig(big.NewInt(int64(expected)))
		assert.False(t, overflow)
		assert.Equal(t, expected, n.Uint64())
	})
}

func TestUint256Serialization(t *testing.T) {
	t.Run("JSON", func(t *testing.T) {
		origin := "123456789123456789123456789"
		n, _ := num.UintFromString(origin, 10)

		// Serialize.
		serialized, err := n.MarshalJSON()
		require.NoError(t, err)

		// Deserialize.
		require.NoError(t, n.UnmarshalJSON(serialized))
		assert.Equal(t, origin, n.String())
	})

	t.Run("Binary", func(t *testing.T) {
		origin := "123456789123456789123456789"
		n, _ := num.UintFromString(origin, 10)

		// Serialize.
		serialized, err := n.MarshalBinary()
		require.NoError(t, err)

		// Deserialize.
		require.NoError(t, n.UnmarshalBinary(serialized))
		assert.Equal(t, origin, n.String())
	})

	t.Run("Database", func(t *testing.T) {
		origin := "123456789123456789123456789"
		n, _ := num.UintFromString(origin, 10)

		// Serialize.
		serialized, err := n.Value()
		require.NoError(t, err)

		// Deserialize.
		require.NoError(t, n.Scan(serialized))
		assert.Equal(t, origin, n.String())
	})
}

func TestUint256Clone(t *testing.T) {
	var (
		expect1 uint64 = 42
		expect2 uint64 = 84
		first          = num.NewUint(expect1)
		second         = first.Clone()
	)

	assert.Equal(t, expect1, first.Uint64())
	assert.Equal(t, expect1, second.Uint64())

	// now we change second value, and ensure 1 hasn't changed
	second.Add(second, num.NewUint(42))

	assert.Equal(t, expect1, first.Uint64())
	assert.Equal(t, expect2, second.Uint64())
}

func TestUint256Copy(t *testing.T) {
	var (
		expect1 uint64 = 42
		expect2 uint64 = 84
		first          = num.NewUint(expect1)
		second         = num.NewUint(expect2)
	)

	assert.Equal(t, expect1, first.Uint64())
	assert.Equal(t, expect2, second.Uint64())

	// now we copy first into second
	second.Copy(first)

	// we check that first and second have the same value
	assert.Equal(t, expect1, first.Uint64())
	assert.Equal(t, expect1, second.Uint64())

	// and now we will update first to have expect2 value
	// and make sure it haven't changed second
	first.SetUint64(expect2)
	assert.Equal(t, expect2, first.Uint64())
	assert.Equal(t, expect1, second.Uint64())
}

func TestUInt256IsZero(t *testing.T) {
	zero := num.NewUint(0)
	assert.True(t, zero.IsZero())
}

func TestUint256Print(t *testing.T) {
	var (
		expected = "42"
		n        = num.NewUint(42)
	)

	assert.Equal(t, expected, fmt.Sprintf("%v", n))
}

func TestUint256Delta(t *testing.T) {
	data := []struct {
		x, y, z uint64
		neg     bool
	}{
		{
			x:   1234,
			y:   1230,
			z:   4,
			neg: false,
		},
		{
			x:   1230,
			y:   1234,
			z:   4,
			neg: true,
		},
	}
	for _, set := range data {
		exp := num.NewUint(set.z)
		x, y := num.NewUint(set.x), num.NewUint(set.y)
		got, neg := num.NewUint(0).Delta(x, y)
		assert.Equal(t, exp.String(), got.String())
		assert.Equal(t, set.neg, neg)
	}
}

func TestSum(t *testing.T) {
	data := []struct {
		x, y, z uint64
		exp     uint64
	}{
		{
			x:   1,
			y:   2,
			z:   3,
			exp: 1 + 2 + 3,
		},
		{
			x:   123,
			y:   456,
			z:   789,
			exp: 123 + 456 + 789,
		},
	}
	for _, set := range data {
		x, y, z := num.NewUint(set.x), num.NewUint(set.y), num.NewUint(set.z)
		exp := num.NewUint(set.exp)
		zero := num.NewUint(0)
		fSum := num.Sum(x, y, z)
		assert.Equal(t, exp.String(), fSum.String())
		ptr := zero.AddSum(x, y, z)
		assert.Equal(t, exp.String(), zero.String())
		assert.Equal(t, zero, ptr)
		// compare to manual:
		manual := num.NewUint(0)
		manual = manual.Add(manual, x)
		assert.NotEqual(t, exp.String(), manual.String(), "manual x only")
		manual = manual.Add(manual, y)
		assert.NotEqual(t, exp.String(), manual.String(), "manual x+y only")
		manual = manual.Add(manual, z)
		assert.Equal(t, exp.String(), manual.String())
	}
}

func TestDeferDoCopy(t *testing.T) {
	var (
		expected1 uint64 = 42
		expected2 uint64 = 84
		n1               = num.NewUint(42)
	)

	n2 := *n1

	assert.Equal(t, expected1, n1.Uint64())
	assert.Equal(t, expected1, n2.Uint64())

	n2.SetUint64(expected2)
	assert.Equal(t, expected1, n1.Uint64())
	assert.Equal(t, expected2, n2.Uint64())
}

func TestDeltaI(t *testing.T) {
	n1 := num.NewUint(10)
	n2 := num.NewUint(25)

	r1 := num.UintZero().DeltaI(n1.Clone(), n2.Clone())
	assert.Equal(t, "-15", r1.String())

	r2 := num.UintZero().DeltaI(n2.Clone(), n1.Clone())
	assert.Equal(t, "15", r2.String())
}

func TestMedian(t *testing.T) {
	require.Nil(t, num.Median(nil))
	require.Equal(t, "10", num.Median([]*num.Uint{num.NewUint(10)}).String())
	require.Equal(t, "10", num.Median([]*num.Uint{num.NewUint(10), num.NewUint(5), num.NewUint(17)}).String())
	require.Equal(t, "11", num.Median([]*num.Uint{num.NewUint(10), num.NewUint(5), num.NewUint(12), num.NewUint(17)}).String())
}

func TestSqrt(t *testing.T) {
	n := num.NewUint(123456789)

	rt := n.Sqrt(n)
	assert.Equal(t, "11111.1110605555554406", rt.String())

	rt = n.Sqrt(num.UintZero())
	assert.Equal(t, "0", rt.String())

	rt = n.Sqrt(num.UintOne())
	assert.Equal(t, "1", rt.String())

	n.SqrtInt(n)
	assert.Equal(t, "11111", n.String())
}
