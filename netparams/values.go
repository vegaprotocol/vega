package netparams

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	validators "github.com/mwitkow/go-proto-validators"
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

func (i *Int) ToInt() (int64, error) {
	return i.value, nil
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

func (d *Duration) ToDuration() (time.Duration, error) {
	return d.value, nil
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

type StringRule func(string) error

type String struct {
	*baseValue
	value   string
	rules   []StringRule
	mutable bool
}

func NewString(rules ...StringRule) *String {
	return &String{
		baseValue: &baseValue{},
		rules:     rules,
	}
}

func (s *String) String() string {
	return s.value
}

func StringValidJSON(t interface{}) func(string) error {
	return func(s string) error {
		dec := json.NewDecoder(bytes.NewReader([]byte(s)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(t); err != nil {
			return err
		}
		// kind := reflect.TypeOf(t).Kind()
		// if kind == reflect.Slice {
		arr := reflect.ValueOf(t)
		for i := 0; i < arr.Len(); i++ {
			if err := validators.CallValidatorIfExists(arr.Index(i)); err != nil {
				return err
			}
		}
		return nil

		return validators.CallValidatorIfExists(t)
	}
}

func (s *String) Validate(value string) error {
	if !s.mutable {
		return errors.New("value is not mutable")
	}

	var err error

	for _, fn := range s.rules {
		if newerr := fn(value); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}

	return err
}

func (s *String) Update(value string) error {
	err := s.Validate(value)
	if err == nil {
		s.value = value
	}

	return err
}

func (s *String) Mutable(b bool) *String {
	s.mutable = b
	return s
}

func (s *String) MustUpdate(value string) *String {
	err := s.Update(value)
	if err != nil {
		panic(err)
	}
	return s
}

func (s *String) ToString() (string, error) {
	return s.value, nil
}
