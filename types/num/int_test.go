package num_test

import (
	"math/rand"
	"testing"

	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

func TestInt256Constructors(t *testing.T) {
	// Positive number
	var value1 int64 = 42
	n := num.NewInt(value1)
	assert.Equal(t, uint64(value1), n.U.Uint64())
	assert.Equal(t, true, n.IsPositive())
	assert.Equal(t, false, n.IsNegative())
	assert.Equal(t, false, n.IsZero())

	// Negative number
	var value2 int64 = -42
	n = num.NewInt(value2)
	assert.Equal(t, uint64(-value2), n.U.Uint64())
	assert.Equal(t, false, n.IsPositive())
	assert.Equal(t, true, n.IsNegative())
	assert.Equal(t, false, n.IsZero())

	// Zero
	var value3 int64 = 0
	n = num.NewInt(value3)
	assert.Equal(t, uint64(value3), n.U.Uint64())
	assert.Equal(t, false, n.IsPositive())
	assert.Equal(t, false, n.IsNegative())
	assert.Equal(t, true, n.IsZero())
}

func TestIntFromUint(t *testing.T) {
	n := num.NewUint(100)

	// Test making a positive value
	i := num.IntFromUint(n, true)
	assert.Equal(t, uint64(100), i.U.Uint64())
	assert.Equal(t, true, i.IsPositive())
	assert.Equal(t, false, i.IsNegative())
	assert.Equal(t, false, i.IsZero())

	// Test making a negative value
	i = num.IntFromUint(n, false)
	assert.Equal(t, uint64(100), i.U.Uint64())
	assert.Equal(t, false, i.IsPositive())
	assert.Equal(t, true, i.IsNegative())
	assert.Equal(t, false, i.IsZero())
}

func TestFlipSign(t *testing.T) {
	n := num.NewInt(100)
	assert.Equal(t, uint64(100), n.U.Uint64())
	assert.Equal(t, true, n.IsPositive())
	assert.Equal(t, false, n.IsNegative())
	assert.Equal(t, false, n.IsZero())

	n.FlipSign()
	assert.Equal(t, uint64(100), n.U.Uint64())
	assert.Equal(t, false, n.IsPositive())
	assert.Equal(t, true, n.IsNegative())
	assert.Equal(t, false, n.IsZero())
}

func TestClone(t *testing.T) {
	n := num.NewInt(100)
	n2 := n.Clone()

	n2.FlipSign()
	assert.Equal(t, true, n.IsPositive())
	assert.Equal(t, true, n2.IsNegative())

	n.AddSum(num.NewInt(50))
	assert.Equal(t, uint64(150), n.U.Uint64())
	assert.Equal(t, uint64(100), n2.U.Uint64())
}

func TestGT(t *testing.T) {
	mid := num.NewInt(0)
	low := num.NewInt(-10)
	high := num.NewInt(10)

	assert.Equal(t, true, mid.GT(low))
	assert.Equal(t, false, mid.GT(high))
	assert.Equal(t, false, low.GT(mid))
	assert.Equal(t, false, low.GT(high))
	assert.Equal(t, true, high.GT(mid))
	assert.Equal(t, true, high.GT(low))

	assert.Equal(t, false, mid.GT(mid))
	assert.Equal(t, false, low.GT(low))
	assert.Equal(t, false, high.GT(high))
}

func TestLT(t *testing.T) {
	mid := num.NewInt(0)
	low := num.NewInt(-10)
	high := num.NewInt(10)

	assert.Equal(t, false, mid.LT(low))
	assert.Equal(t, true, mid.LT(high))
	assert.Equal(t, true, low.LT(mid))
	assert.Equal(t, true, low.LT(high))
	assert.Equal(t, false, high.LT(mid))
	assert.Equal(t, false, high.LT(low))

	assert.Equal(t, false, mid.LT(mid))
	assert.Equal(t, false, low.LT(low))
	assert.Equal(t, false, high.LT(high))
}

func TestString(t *testing.T) {
	mid := num.NewInt(0)
	low := num.NewInt(-10)
	high := num.NewInt(10)

	assert.Equal(t, "0", mid.String())
	assert.Equal(t, "-10", low.String())
	assert.Equal(t, "10", high.String())
}

func TestAdd(t *testing.T) {
	// Add positive to zero
	i := num.NewInt(0)
	i.Add(num.NewInt(10))
	assert.Equal(t, "10", i.String())

	// Add negative to zero
	i = num.NewInt(0)
	i.Add(num.NewInt(-10))
	assert.Equal(t, "-10", i.String())

	// Add zero to negative
	i = num.NewInt(0)
	i.Add(num.NewInt(0))
	assert.Equal(t, "0", i.String())

	// Add zero to positive
	i = num.NewInt(10)
	i.Add(num.NewInt(0))
	assert.Equal(t, "10", i.String())

	// Add zero to zero
	i = num.NewInt(0)
	i.Add(num.NewInt(0))
	assert.Equal(t, "0", i.String())

	// Add positive to positive
	i = num.NewInt(10)
	i.Add(num.NewInt(15))
	assert.Equal(t, "25", i.String())

	// Add negative to negative
	i = num.NewInt(-10)
	i.Add(num.NewInt(-15))
	assert.Equal(t, "-25", i.String())

	// Add positive to negative (no sign flip)
	i = num.NewInt(-15)
	i.Add(num.NewInt(10))
	assert.Equal(t, "-5", i.String())

	// Add positive to negative (sign flip)
	i = num.NewInt(-10)
	i.Add(num.NewInt(15))
	assert.Equal(t, "5", i.String())

	// Add negative to positive (no sign flip)
	i = num.NewInt(10)
	i.Add(num.NewInt(-5))
	assert.Equal(t, "5", i.String())

	// Add negative to positive (sign flip)
	i = num.NewInt(10)
	i.Add(num.NewInt(-15))
	assert.Equal(t, "-5", i.String())
}

func TestAddSum(t *testing.T) {
	num1 := num.NewInt(10)
	num2 := num.NewInt(20)
	num3 := num.NewInt(-15)
	num4 := num.NewInt(-30)
	num5 := num.NewInt(10)

	result := num1.AddSum(num2, num3, num4, num5)
	assert.Equal(t, "-5", result.String())
}

func TestSubSum(t *testing.T) {
	num1 := num.NewInt(10)
	num2 := num.NewInt(20)
	num3 := num.NewInt(-15)
	num4 := num.NewInt(-30)
	num5 := num.NewInt(10)

	result := num1.SubSum(num2, num3, num4, num5)
	assert.Equal(t, "25", result.String())
}

func TestBruteForce(t *testing.T) {
	t.Run("brute force adds", testAddLoop)
	t.Run("brute force subs", testSubLoop)
}

func testAddLoop(t *testing.T) {
	for c := 0; c < 10000; c++ {
		num1 := rand.Int63n(100) - 50
		num2 := rand.Int63n(100) - 50

		bigNum1 := num.NewInt(num1)
		bigNum2 := num.NewInt(num2)

		bigNum1.Add(bigNum2)

		assert.Equal(t, num1+num2, bigNum1.Int64())
		// fmt.Println(num1, num2, num1-num2, bigNum1.String())
	}
}

func testSubLoop(t *testing.T) {
	for c := 0; c < 10000; c++ {
		num1 := rand.Int63n(100) - 50
		num2 := rand.Int63n(100) - 50

		bigNum1 := num.NewInt(num1)
		bigNum2 := num.NewInt(num2)

		bigNum1.Sub(bigNum2)

		assert.Equal(t, num1-num2, bigNum1.Int64())
		// fmt.Println(num1, num2, num1-num2, bigNum1.String())
	}
}
