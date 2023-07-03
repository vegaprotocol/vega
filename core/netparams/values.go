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

package netparams

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
)

type baseValue struct{}

func (b *baseValue) ToDecimal() (num.Decimal, error) {
	return num.DecimalZero(), errors.New("not a decimal value")
}

func (b *baseValue) ToInt() (int64, error) {
	return 0, errors.New("not an int value")
}

func (b *baseValue) ToUint() (*num.Uint, error) {
	return num.UintZero(), errors.New("not an uint value")
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

type DecimalRule func(num.Decimal) error

type Decimal struct {
	*baseValue
	value   num.Decimal
	rawval  string
	rules   []DecimalRule
	mutable bool
}

func NewDecimal(rules ...DecimalRule) *Decimal {
	return &Decimal{
		baseValue: &baseValue{},
		rules:     rules,
	}
}

func (f *Decimal) GetDispatch() func(context.Context, interface{}) error {
	return func(ctx context.Context, rawfn interface{}) error {
		// there can't be errors here, as all dispatcher
		// should have been check earlier when being register
		fn := rawfn.(func(context.Context, num.Decimal) error)
		return fn(ctx, f.value)
	}
}

func (f *Decimal) CheckDispatch(fn interface{}) error {
	if _, ok := fn.(func(context.Context, num.Decimal) error); !ok {
		return errors.New("invalid type, expected func(context.Context, float64) error")
	}
	return nil
}

func (f *Decimal) AddRules(fns ...interface{}) error {
	for _, fn := range fns {
		// asset they have the right type
		v, ok := fn.(DecimalRule)
		if !ok {
			return errors.New("floats require DecimalRule functions")
		}
		f.rules = append(f.rules, v)
	}

	return nil
}

func (f *Decimal) ToDecimal() (num.Decimal, error) {
	return f.value, nil
}

func (f *Decimal) Validate(value string) error {
	valf, err := num.DecimalFromString(value)
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

func (f *Decimal) UpdateOptionalValidation(value string, validate bool) error {
	if !f.mutable {
		return errors.New("value is not mutable")
	}
	valf, err := num.DecimalFromString(value)
	if err != nil {
		return err
	}

	if validate {
		if err := f.Validate(value); err != nil {
			return err
		}
	}

	f.rawval = value
	f.value = valf
	return nil
}

func (f *Decimal) Update(value string) error {
	return f.UpdateOptionalValidation(value, true)
}

func (f *Decimal) Mutable(b bool) *Decimal {
	f.mutable = b
	return f
}

func (f *Decimal) MustUpdate(value string) *Decimal {
	if err := f.Update(value); err != nil {
		panic(err)
	}
	return f
}

func (f *Decimal) String() string {
	return f.rawval
}

func DecimalGTE(f num.Decimal) func(num.Decimal) error {
	return func(val num.Decimal) error {
		if val.GreaterThanOrEqual(f) {
			return nil
		}
		return fmt.Errorf("expect >= %v got %v", f, val)
	}
}

func DecimalDependentLT(otherName string, other *Decimal) func(num.Decimal) error {
	return func(val num.Decimal) error {
		if val.LessThan(other.value) {
			return nil
		}
		return fmt.Errorf("expect < %v (%s) got %v", other.value, otherName, val)
	}
}

func DecimalDependentLTE(otherName string, other *Decimal) func(num.Decimal) error {
	return func(val num.Decimal) error {
		if val.LessThanOrEqual(other.value) {
			return nil
		}
		return fmt.Errorf("expect <= %v (%s) got %v", other.value, otherName, val)
	}
}

func DecimalGT(f num.Decimal) func(num.Decimal) error {
	return func(val num.Decimal) error {
		if val.GreaterThan(f) {
			return nil
		}
		return fmt.Errorf("expect > %v got %v", f, val)
	}
}

func DecimalLTE(f num.Decimal) func(num.Decimal) error {
	return func(val num.Decimal) error {
		if val.LessThanOrEqual(f) {
			return nil
		}
		return fmt.Errorf("expect <= %v got %v", f, val)
	}
}

func DecimalLT(f num.Decimal) func(num.Decimal) error {
	return func(val num.Decimal) error {
		if val.LessThan(f) {
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

func (i *Int) UpdateOptionalValidation(value string, validate bool) error {
	if !i.mutable {
		return errors.New("value is not mutable")
	}
	vali, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}

	if validate {
		if err := i.Validate(value); err != nil {
			return err
		}
	}

	i.rawval = value
	i.value = vali

	return nil
}

func (i *Int) Update(value string) error {
	return i.UpdateOptionalValidation(value, true)
}

func (i *Int) Mutable(b bool) *Int {
	i.mutable = b
	return i
}

func (i *Int) MustUpdate(value string) *Int {
	if err := i.Update(value); err != nil {
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

type TimeRule func(time.Time) error

type Time struct {
	*baseValue
	value   time.Time
	rawval  string
	rules   []TimeRule
	mutable bool
}

func NewTime(rules ...TimeRule) *Time {
	return &Time{
		baseValue: &baseValue{},
		rules:     rules,
	}
}

func (t *Time) GetDispatch() func(context.Context, interface{}) error {
	return func(ctx context.Context, rawfn interface{}) error {
		// there can't be errors here, as all dispatcher
		// should have been check earlier when being register
		fn := rawfn.(func(context.Context, time.Time) error)
		return fn(ctx, t.value)
	}
}

func (t *Time) CheckDispatch(fn interface{}) error {
	if _, ok := fn.(func(context.Context, time.Time) error); !ok {
		return errors.New("invalid type, expected func(context.Context, time.Time) error")
	}
	return nil
}

func (t *Time) AddRules(fns ...interface{}) error {
	for _, fn := range fns {
		// asset they have the right type
		v, ok := fn.(TimeRule)
		if !ok {
			return errors.New("times require TimeRule functions")
		}
		t.rules = append(t.rules, v)
	}

	return nil
}

func (t *Time) ToTime() (time.Time, error) {
	return t.value, nil
}

func (t *Time) Validate(value string) error {
	if !t.mutable {
		return errors.New("value is not mutable")
	}
	pVal, err := parseTime(value)
	if err != nil {
		return err
	}
	tVal := *pVal
	for _, fn := range t.rules {
		if newerr := fn(tVal); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}
	return err
}

func (t *Time) Update(value string) error {
	if !t.mutable {
		return errors.New("value is not mutable")
	}
	pVal, err := parseTime(value)
	if err != nil {
		return err
	}
	tVal := *pVal
	for _, fn := range t.rules {
		if newerr := fn(tVal); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}

	if err == nil {
		t.rawval = value
		t.value = tVal
	}
	return nil
}

func (t *Time) Mutable(b bool) *Time {
	t.mutable = b
	return t
}

func (t *Time) MustUpdate(value string) *Time {
	if err := t.Update(value); err != nil {
		panic(err)
	}
	return t
}

func (t *Time) String() string {
	return t.rawval
}

func TimeNonZero() func(time.Time) error {
	return func(val time.Time) error {
		if !val.IsZero() {
			return nil
		}
		return fmt.Errorf("expect non-zero time")
	}
}

func parseTime(v string) (*time.Time, error) {
	if v == "never" {
		return &time.Time{}, nil
	}
	formats := []string{
		time.RFC3339,
		"2006-01-02",
	}
	for _, f := range formats {
		if tVal, err := time.Parse(f, v); err == nil {
			return &tVal, nil
		}
	}
	// last attempt -> timestamp
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil, err
	}
	t := time.Unix(i, 0)
	return &t, nil
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
	return d.UpdateOptionalValidation(value, true)
}

func (d *Duration) UpdateOptionalValidation(value string, validate bool) error {
	if !d.mutable {
		return errors.New("value is not mutable")
	}
	vali, err := time.ParseDuration(value)
	if err != nil {
		return err
	}

	if validate {
		if err := d.Validate(value); err != nil {
			return err
		}
	}

	d.rawval = value
	d.value = vali
	return nil
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

func DurationDependentGT(otherName string, other *Duration) DurationRule {
	return func(val time.Duration) error {
		if val > other.value {
			return nil
		}
		return fmt.Errorf("expect > %v (%s) got %v", other.value, otherName, val)
	}
}

func DurationDependentGTE(otherName string, other *Duration) DurationRule {
	return func(val time.Duration) error {
		if val >= other.value {
			return nil
		}
		return fmt.Errorf("expect >= %v (%s) got %v", other.value, otherName, val)
	}
}

func DurationDependentLT(otherName string, other *Duration) DurationRule {
	return func(val time.Duration) error {
		if val < other.value {
			return nil
		}
		return fmt.Errorf("expect < %v (%s) got %v", other.value, otherName, val)
	}
}

func DurationDependentLTE(otherName string, other *Duration) DurationRule {
	return func(val time.Duration) error {
		if val <= other.value {
			return nil
		}
		return fmt.Errorf("expect <= %v (%s) got %v", other.value, otherName, val)
	}
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
	d := json.NewDecoder(bytes.NewReader(value))
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

func (j *JSON) UpdateOptionalValidation(value string, validate bool) error {
	if !j.mutable {
		return errors.New("value is not mutable")
	}
	if validate {
		if err := j.Validate(value); err != nil {
			return err
		}
	}

	j.rawval = value
	return nil
}

func (j *JSON) Update(value string) error {
	return j.UpdateOptionalValidation(value, true)
}

func (j *JSON) Mutable(b bool) *JSON {
	j.mutable = b
	return j
}

func (j *JSON) MustUpdate(value string) *JSON {
	if err := j.Update(value); err != nil {
		panic(err)
	}
	return j
}

func (j *JSON) String() string {
	return j.rawval
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

func (s *String) UpdateOptionalValidation(value string, validate bool) error {
	if !s.mutable {
		return errors.New("value is not mutable")
	}

	if validate {
		if err := s.Validate(value); err != nil {
			return err
		}
	}

	s.rawval = value
	return nil
}

func (s *String) Update(value string) error {
	return s.UpdateOptionalValidation(value, true)
}

func (s *String) Mutable(b bool) *String {
	s.mutable = b
	return s
}

func (s *String) MustUpdate(value string) *String {
	if err := s.Update(value); err != nil {
		panic(err)
	}
	return s
}

func (s *String) String() string {
	return s.rawval
}

type UintRule func(*num.Uint) error

type Uint struct {
	*baseValue
	value   *num.Uint
	rawval  string
	rules   []UintRule
	mutable bool
}

func NewUint(rules ...UintRule) *Uint {
	return &Uint{
		baseValue: &baseValue{},
		rules:     rules,
		value:     num.UintZero(),
	}
}

func (i *Uint) GetDispatch() func(context.Context, interface{}) error {
	return func(ctx context.Context, rawfn interface{}) error {
		// there can't be errors here, as all dispatcher
		// should have been check earlier when being register
		fn := rawfn.(func(context.Context, *num.Uint) error)
		return fn(ctx, i.value.Clone())
	}
}

func (i *Uint) CheckDispatch(fn interface{}) error {
	if _, ok := fn.(func(context.Context, *num.Uint) error); !ok {
		return errors.New("invalid type, expected func(context.Context, *num.Uint) error")
	}
	return nil
}

func (i *Uint) AddRules(fns ...interface{}) error {
	for _, fn := range fns {
		// asset they have the right type
		v, ok := fn.(UintRule)
		if !ok {
			return errors.New("ints require BigUintRule functions")
		}
		i.rules = append(i.rules, v)
	}

	return nil
}

func (i *Uint) ToUint() (*num.Uint, error) {
	return i.value.Clone(), nil
}

func (i *Uint) Validate(value string) error {
	if strings.HasPrefix(strings.TrimLeft(value, " "), "-") {
		return errors.New("invalid uint")
	}

	val, overflow := num.UintFromString(value, 10)
	if overflow {
		return errors.New("invalid uint")
	}

	if !i.mutable {
		return errors.New("value is not mutable")
	}

	var err error
	for _, fn := range i.rules {
		if newerr := fn(val.Clone()); newerr != nil {
			if err != nil {
				err = fmt.Errorf("%v, %w", err, newerr)
			} else {
				err = newerr
			}
		}
	}
	return err
}

func (i *Uint) UpdateOptionalValidation(value string, validate bool) error {
	if !i.mutable {
		return errors.New("value is not mutable")
	}
	if strings.HasPrefix(strings.TrimLeft(value, " "), "-") {
		return errors.New("invalid uint")
	}

	val, overflow := num.UintFromString(value, 10)
	if overflow {
		return errors.New("invalid uint")
	}

	if validate {
		if err := i.Validate(value); err != nil {
			return err
		}
	}

	i.rawval = value
	i.value = val

	return nil
}

func (i *Uint) Update(value string) error {
	return i.UpdateOptionalValidation(value, true)
}

func (i *Uint) Mutable(b bool) *Uint {
	i.mutable = b
	return i
}

func (i *Uint) MustUpdate(value string) *Uint {
	if err := i.Update(value); err != nil {
		panic(err)
	}
	return i
}

func (i *Uint) String() string {
	return i.rawval
}

func UintGTE(i *num.Uint) func(*num.Uint) error {
	icopy := i.Clone()
	return func(val *num.Uint) error {
		if val.GTE(icopy) {
			return nil
		}
		return fmt.Errorf("expect >= %v got %v", i, val)
	}
}

// ensure that the value is >= the other value x factor.
func UintDependentGTE(otherName string, other *Uint, factor num.Decimal) UintRule {
	return func(val *num.Uint) error {
		lowerBound, _ := num.UintFromDecimal(other.value.ToDecimal().Mul(factor))
		if val.GTE(lowerBound) {
			return nil
		}
		return fmt.Errorf("expect >= %v (%s * %s) got %v", lowerBound, otherName, factor.String(), val)
	}
}

// ensure that the value is <= the other value x factor.
func UintDependentLTE(otherName string, other *Uint, factor num.Decimal) UintRule {
	return func(val *num.Uint) error {
		upperBound, _ := num.UintFromDecimal(other.value.ToDecimal().Mul(factor))
		if val.LTE(upperBound) {
			return nil
		}
		return fmt.Errorf("expect <= %v (%s * %s) got %v", upperBound, otherName, factor.String(), val)
	}
}

func UintGT(i *num.Uint) func(*num.Uint) error {
	icopy := i.Clone()
	return func(val *num.Uint) error {
		if val.GT(icopy) {
			return nil
		}
		return fmt.Errorf("expect > %v got %v", i, val)
	}
}

func UintLTE(i *num.Uint) func(*num.Uint) error {
	icopy := i.Clone()
	return func(val *num.Uint) error {
		if val.LTE(icopy) {
			return nil
		}
		return fmt.Errorf("expect <= %v got %v", i, val)
	}
}

func UintLT(i *num.Uint) func(*num.Uint) error {
	icopy := i.Clone()
	return func(val *num.Uint) error {
		if val.LT(icopy) {
			return nil
		}
		return fmt.Errorf("expect < %v got %v", i, val)
	}
}
