package netparams

import (
	"bytes"
	"context"
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

func (b *baseValue) ToJSONStruct(v Reset) error {
	return errors.New("not a JSON value")
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

func (f *Float) GetDispatch() func(context.Context, interface{}) error {
	return func(ctx context.Context, rawfn interface{}) error {
		// there can't be errors here, as all dispatcher
		// should have been check earlier when being register
		fn := rawfn.(func(context.Context, float64) error)
		return fn(ctx, f.value)
	}
}

func (f *Float) CheckDispatch(fn interface{}) error {
	if _, ok := fn.(func(context.Context, float64) error); !ok {
		return errors.New("invalid type, expected func(context.Context, float64) error")
	}
	return nil
}

func (f *Float) AddRules(fns ...interface{}) error {
	for _, fn := range fns {
		// asset they have the right type
		v, ok := fn.(FloatRule)
		if !ok {
			return errors.New("floats require FloatRule functions")
		}
		f.rules = append(f.rules, v)
	}

	return nil
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

func (i *Int) GetDispatch() func(context.Context, interface{}) error {
	return func(ctx context.Context, rawfn interface{}) error {
		// there can't be errors here, as all dispatcher
		// should have been check earlier when being register
		fn := rawfn.(func(context.Context, int64) error)
		return fn(ctx, i.value)
	}
}

func (i *Int) CheckDispatch(fn interface{}) error {
	if _, ok := fn.(func(context.Context, int64) error); !ok {
		return errors.New("invalid type, expected func(context.Context, int64) error")
	}
	return nil
}

func (i *Int) AddRules(fns ...interface{}) error {
	for _, fn := range fns {
		// asset they have the right type
		v, ok := fn.(IntRule)
		if !ok {
			return errors.New("ints require IntRule functions")
		}
		i.rules = append(i.rules, v)
	}

	return nil
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

func (d *Duration) GetDispatch() func(context.Context, interface{}) error {
	return func(ctx context.Context, rawfn interface{}) error {
		// there can't be errors here, as all dispatcher
		// should have been check earlier when being register
		fn := rawfn.(func(context.Context, time.Duration) error)
		return fn(ctx, d.value)
	}
}

func (d *Duration) CheckDispatch(fn interface{}) error {
	if _, ok := fn.(func(context.Context, time.Duration) error); !ok {
		return errors.New("invalid type, expected func(context.Context, time.Duration) error")
	}
	return nil
}

func (d *Duration) AddRules(fns ...interface{}) error {
	for _, fn := range fns {
		// asset they have the right type
		v, ok := fn.(DurationRule)
		if !ok {
			return errors.New("durations require DurationRule functions")
		}
		d.rules = append(d.rules, v)
	}

	return nil
}

func (d *Duration) ToDuration() (time.Duration, error) {
	return d.value, nil
}

func (d *Duration) Validate(value string) error {
	vali, err := time.ParseDuration(value)
	if err != nil {
		return err
	}

	if !d.mutable {
		return errors.New("value is not mutable")
	}

	for _, fn := range d.rules {
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

func (d *Duration) Update(value string) error {
	if !d.mutable {
		return errors.New("value is not mutable")
	}
	vali, err := time.ParseDuration(value)
	if err != nil {
		return err
	}

	for _, fn := range d.rules {
		if newerr := fn(vali); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}

	if err == nil {
		d.rawval = value
		d.value = vali
	}

	return err
}

func (d *Duration) Mutable(b bool) *Duration {
	d.mutable = b
	return d
}

func (d *Duration) MustUpdate(value string) *Duration {
	if err := d.Update(value); err != nil {
		panic(err)
	}
	return d
}

func (d *Duration) String() string {
	return d.rawval
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

type JSONRule func(interface{}) error

type JSON struct {
	*baseValue
	value   Reset
	ty      reflect.Type
	rawval  string
	rules   []JSONRule
	mutable bool
}

func NewJSON(val Reset, rules ...JSONRule) *JSON {
	if val == nil {
		panic("JSON values requires non nil pointers")
	}
	ty := reflect.TypeOf(val)
	if ty.Kind() != reflect.Ptr {
		panic("JSON values requires pointers")
	}
	return &JSON{
		baseValue: &baseValue{},
		rules:     rules,
		ty:        ty,
		value:     val,
	}

}

func (j *JSON) ToJSONStruct(v Reset) error {
	if v == nil {
		return errors.New("nil interface{}")
	}
	// just make sure types are compatible
	if !reflect.TypeOf(v).AssignableTo(j.ty) {
		return errors.New("incompatible type")
	}

	return json.Unmarshal([]byte(j.rawval), v)
}

func (j *JSON) GetDispatch() func(context.Context, interface{}) error {
	return func(ctx context.Context, rawfn interface{}) error {
		// there can't be errors here, as all dispatcher
		// should have been check earlier when being register
		fn := rawfn.(func(context.Context, interface{}) error)
		json.Unmarshal([]byte(j.rawval), j.value)
		return fn(ctx, j.value)
	}
}

func (j *JSON) CheckDispatch(fn interface{}) error {
	if _, ok := fn.(func(context.Context, interface{}) error); !ok {
		return errors.New("invalid type, expected func(context.Context, float64) error")
	}
	return nil
}

func (j *JSON) AddRules(fns ...interface{}) error {
	for _, fn := range fns {
		fmt.Printf("JSONRULE: %#v\n", fn)
		// asset they have the right type
		v, ok := fn.(JSONRule)
		if !ok {
			return errors.New("JSONs require JSONRule functions")
		}
		j.rules = append(j.rules, v)
	}

	return nil
}

func (j *JSON) validateValue(value []byte) error {
	j.value.Reset()
	d := json.NewDecoder(bytes.NewReader([]byte(value)))
	d.DisallowUnknownFields()
	return d.Decode(j.value)
}

func (j *JSON) Validate(value string) error {
	err := j.validateValue([]byte(value))
	if err != nil {
		return fmt.Errorf("unable to unmarshal value, %w", err)
	}

	if !j.mutable {
		return errors.New("value is not mutable")
	}

	for _, fn := range j.rules {
		if newerr := fn(j.value); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}

	return err
}

func (j *JSON) Update(value string) error {
	err := j.validateValue([]byte(value))
	if err != nil {
		return fmt.Errorf("unable to unmarshal value, %w", err)
	}

	if !j.mutable {
		return errors.New("value is not mutable")
	}

	for _, fn := range j.rules {
		if newerr := fn(j.value); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}

	if err != nil {
		return err
	}

	j.rawval = value
	return nil
}

func (j *JSON) Mutable(b bool) *JSON {
	j.mutable = b
	return j
}

func (j *JSON) MustUpdate(value string) *JSON {
	err := j.Update(value)
	if err != nil {
		panic(err)
	}
	return j
}

func (j *JSON) String() string {
	return j.rawval
}

func JSONProtoValidator() func(interface{}) error {
	return func(t interface{}) error {
		return validators.CallValidatorIfExists(t)
	}
}

type StringRule func(string) error

type String struct {
	*baseValue
	rawval  string
	rules   []StringRule
	mutable bool
}

func NewString(rules ...StringRule) *String {
	return &String{
		baseValue: &baseValue{},
		rules:     rules,
	}
}

func (s *String) GetDispatch() func(context.Context, interface{}) error {
	return func(ctx context.Context, rawfn interface{}) error {
		// there can't be errors here, as all dispatcher
		// should have been check earlier when being register
		fn := rawfn.(func(context.Context, string) error)
		return fn(ctx, s.rawval)
	}
}

func (s *String) CheckDispatch(fn interface{}) error {
	if _, ok := fn.(func(context.Context, string) error); !ok {
		return errors.New("invalid type, expected func(context.Context, string) error")
	}
	return nil
}

func (s *String) AddRules(fns ...interface{}) error {
	for _, fn := range fns {
		// asset they have the right type
		v, ok := fn.(StringRule)
		if !ok {
			fmt.Printf("v: %#v\n", v)
			return errors.New("strings require StringRule functions")
		}
		s.rules = append(s.rules, v)
	}

	return nil
}

func (s *String) ToString() (string, error) {
	return s.rawval, nil
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

	if err == nil {
		s.rawval = value
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

func (s *String) String() string {
	return s.rawval
}
