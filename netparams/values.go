package netparams

import (
	"errors"
	"fmt"
	"strconv"
	"time"
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

func (b *baseValue) ToDuration() (time.Duration, error) {
	return 0, errors.New("not a time.Duration value")
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

	if !f.mutable {
		return errors.New("value is not mutable")
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

type IntRule func(int64) error

type Int struct {
	*baseValue
	value   int64
	rawval  string
	rules   []IntRule
	mutable bool
}

func NewInt(rules ...IntRule) *Int {
	return &Int{
		baseValue: &baseValue{},
		rules:     rules,
	}
}

func (i *Int) Validate(value string) error {
	vali, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}

	if !i.mutable {
		return errors.New("value is not mutable")
	}

	for _, fn := range i.rules {
		if newerr := fn(vali); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}
	return err
}

func (i *Int) Update(value string) error {
	if !i.mutable {
		return errors.New("value is not mutable")
	}
	vali, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}

	for _, fn := range i.rules {
		if newerr := fn(vali); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}

	if err == nil {
		i.rawval = value
		i.value = vali
	}

	return err
}

func (i *Int) Mutable(b bool) *Int {
	i.mutable = b
	return i
}

func (i *Int) MustUpdate(value string) *Int {
	err := i.Update(value)
	if err != nil {
		panic(err)
	}
	return i
}

func (i *Int) String() string {
	return i.rawval
}

func IntGTE(i int64) func(int64) error {
	return func(val int64) error {
		if val >= i {
			return nil
		}
		return fmt.Errorf("expect >= %v got %v", i, val)
	}
}

func IntGT(i int64) func(int64) error {
	return func(val int64) error {
		if val > i {
			return nil
		}
		return fmt.Errorf("expect > %v got %v", i, val)
	}
}

func IntLTE(i int64) func(int64) error {
	return func(val int64) error {
		if val <= i {
			return nil
		}
		return fmt.Errorf("expect <= %v got %v", i, val)
	}
}

func IntLT(i int64) func(int64) error {
	return func(val int64) error {
		if val < i {
			return nil
		}
		return fmt.Errorf("expect < %v got %v", i, val)
	}
}

type DurationRule func(time.Duration) error

type Duration struct {
	*baseValue
	value   time.Duration
	rawval  string
	rules   []DurationRule
	mutable bool
}

func NewDuration(rules ...DurationRule) *Duration {
	return &Duration{
		baseValue: &baseValue{},
		rules:     rules,
	}
}

func (i *Duration) Validate(value string) error {
	vali, err := time.ParseDuration(value)
	if err != nil {
		return err
	}

	if !i.mutable {
		return errors.New("value is not mutable")
	}

	for _, fn := range i.rules {
		if newerr := fn(vali); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}
	return err
}

func (i *Duration) Update(value string) error {
	if !i.mutable {
		return errors.New("value is not mutable")
	}
	vali, err := time.ParseDuration(value)
	if err != nil {
		return err
	}

	for _, fn := range i.rules {
		if newerr := fn(vali); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}

	if err == nil {
		i.rawval = value
		i.value = vali
	}

	return err
}

func (i *Duration) Mutable(b bool) *Duration {
	i.mutable = b
	return i
}

func (i *Duration) MustUpdate(value string) *Duration {
	err := i.Update(value)
	if err != nil {
		panic(err)
	}
	return i
}

func (i *Duration) String() string {
	return i.rawval
}

func DurationGTE(i time.Duration) func(time.Duration) error {
	return func(val time.Duration) error {
		if val >= i {
			return nil
		}
		return fmt.Errorf("expect >= %v got %v", i, val)
	}
}

func DurationGT(i time.Duration) func(time.Duration) error {
	return func(val time.Duration) error {
		if val > i {
			return nil
		}
		return fmt.Errorf("expect > %v got %v", i, val)
	}
}

func DurationLTE(i time.Duration) func(time.Duration) error {
	return func(val time.Duration) error {
		if val <= i {
			return nil
		}
		return fmt.Errorf("expect <= %v got %v", i, val)
	}
}

func DurationLT(i time.Duration) func(time.Duration) error {
	return func(val time.Duration) error {
		if val < i {
			return nil
		}
		return fmt.Errorf("expect < %v got %v", i, val)
	}
}
