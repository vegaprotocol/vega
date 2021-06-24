package num_test

import (
	"fmt"
	"math/big"
	"testing"

	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

func TestUint256Constructors(t *testing.T) {
	var expected uint64 = 42

	t.Run("test from uint64", func(t *testing.T) {
		n := num.NewUint(expected)
		assert.Equal(t, expected, n.Uint64())
	})

	t.Run("test from string", func(t *testing.T) {
		n, ok := num.UintFromString("42", 10)
		assert.False(t, ok)
		assert.Equal(t, expected, n.Uint64())
	})

	t.Run("test from big", func(t *testing.T) {
		n, ok := num.UintFromBig(big.NewInt(int64(expected)))
		assert.False(t, ok)
		assert.Equal(t, expected, n.Uint64())
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
