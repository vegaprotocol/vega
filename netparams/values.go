package netparams

import (
	"errors"
	"fmt"
	"strconv"
)

type baseValue struct{}

func (b *baseValue) ToFloat() (float64, error) {
	return 0, errors.New("not a float value")
}

func (b *baseValue) ToInt() (int64, error) {
	return 0, errors.New("not an int value")
}

func (b *baseValue) ToUint() (uint64, error) {
	return 0, errors.New("not an uint value")
}

func (b *baseValue) ToBool() (bool, error) {
	return false, errors.New("not a bool value")
}

func (b *baseValue) ToString() (string, error) {
	return "", errors.New("not a string value")
}

type FloatRule func(float64) error

type Float struct {
	*baseValue
	value   float64
	rawval  string
	rules   []FloatRule
	mutable bool
}

func NewFloat(rules ...FloatRule) *Float {
	return &Float{
		baseValue: &baseValue{},
		rules:     rules,
	}
}

func (f *Float) ToFloat() (float64, error) {
	return f.value, nil
}

func (f *Float) Validate(value string) error {
	valf, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}

	for _, fn := range f.rules {
		if newerr := fn(valf); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}
	return err
}

func (f *Float) Update(value string) error {
	if !f.mutable {
		return errors.New("value is not mutable")
	}
	valf, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}

	for _, fn := range f.rules {
		if newerr := fn(valf); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}

	if err == nil {
		f.rawval = value
		f.value = valf
	}

	return err
}

func (f *Float) Mutable(b bool) *Float {
	f.mutable = b
	return f
}

func (f *Float) MustUpdate(value string) *Float {
	err := f.Update(value)
	if err != nil {
		panic(err)
	}
	return f
}

func (f *Float) String() string {
	return f.rawval
}

func FloatGTE(f float64) func(float64) error {
	return func(val float64) error {
		if val >= f {
			return nil
		}
		return fmt.Errorf("expect >= %v got %v", f, val)
	}
}

func FloatGT(f float64) func(float64) error {
	return func(val float64) error {
		if val > f {
			return nil
		}
		return fmt.Errorf("expect > %v got %v", f, val)
	}
}

func FloatLTE(f float64) func(float64) error {
	return func(val float64) error {
		if val <= f {
			return nil
		}
		return fmt.Errorf("expect <= %v got %v", f, val)
	}
}

func FloatLT(f float64) func(float64) error {
	return func(val float64) error {
		if val < f {
			return nil
		}
		return fmt.Errorf("expect < %v got %v", f, val)
	}
}
