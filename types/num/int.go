package num

// Int a wrapper to a signed big int
type Int struct {
	// The unsigned version of the integer
	U *Uint
	// The sign of the integer true = positive, false = negative
	s bool
}

func IntFromUint(u *Uint, s bool) *Int {
	copy := &Int{s: s,
		U: u.Clone()}
	return copy
}

// IsNegative tests if the stored value is negative
// true if < 0
// false if >= 0
func (i *Int) IsNegative() bool {
	return !i.s && !i.U.IsZero()
}

// IsPositive tests if the stored value is positive
// true if > 0
// false if <= 0
func (i *Int) IsPositive() bool {
	return i.s && !i.U.IsZero()
}

// IsZero tests if the stored value is zero
// true if == 0
func (i *Int) IsZero() bool {
	return i.U.IsZero()
}

// FlipSign changes the sign of the number from - to + and back again.
func (i *Int) FlipSign() {
	i.s = !i.s
}

// Clone creates a copy of the object so nothing is shared
func (i Int) Clone() *Int {
	return &Int{U: i.U.Clone(),
		s: i.s}
}

// GT returns if i > o
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

// LT returns if i < o
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

func (i Int) Int64() int64 {
	val := int64(i.U.Uint64())
	if i.IsNegative() {
		return -val
	}
	return val
}

// String returns a string version of the number
func (i Int) String() string {
	val := i.U.String()
	if i.IsNegative() {
		return "-" + val
	}
	return val
}

// Add will add the passed in value to the base value
// i = i + a
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
// i = i - a
func (i *Int) Sub(a *Int) *Int {
	a.FlipSign()
	i.Add(a)
	a.FlipSign()
	return i
}

// AddSum adds all of the parameters to i
// i = i + a + b + c
func (i *Int) AddSum(vals ...*Int) *Int {
	for _, x := range vals {
		i.Add(x)
	}
	return i
}

// SubSum subtracts all of the parameters from i
// i = i - a - b - c
func (i *Int) SubSum(vals ...*Int) *Int {
	for _, x := range vals {
		i.Sub(x)
	}
	return i
}

// NewInt creates a new Int with the value of the
// int64 passed as a parameter.
func NewInt(val int64) *Int {
	if val < 0 {
		return &Int{U: NewUint(uint64(-val)),
			s: false}
	}
	return &Int{U: NewUint(uint64(val)),
		s: true}
}

// NewIntFromUint creates a new Int with the value of the
// uint passed as a parameter.
func NewIntFromUint(val *Uint) *Int {
	return &Int{U: val,
		s: true}
}
