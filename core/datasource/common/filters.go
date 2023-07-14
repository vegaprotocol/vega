// Copyright (c) 2023 Gobalsky Labs Limited
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

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package common

import (
	"fmt"
	"strconv"

	errors "code.vegaprotocol.io/vega/core/datasource/errors"
	"code.vegaprotocol.io/vega/libs/num"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

type filter struct {
	propertyName     string
	propertyType     datapb.PropertyKey_Type
	numberOfDecimals uint64
	conditions       []condition
}

type condition func(string) (bool, error)

type Filters struct {
	filters map[string]filter
}

func NewFilters(filtersFromSpec []*SpecFilter, isExtType bool) (Filters, error) {
	typedFilters := map[string]filter{}
	for _, f := range filtersFromSpec {
		if isExtType {
			if SpecPropertyKeyIsEmpty(f.Key) {
				return Filters{}, errors.ErrMissingPropertyKey
			}

			_, exist := typedFilters[f.Key.Name]
			if exist {
				return Filters{}, errors.ErrDataSourceSpecHasMultipleSameKeyNamesInFilterList
			}

			for _, condition := range f.Conditions {
				if f.Key.Type == datapb.PropertyKey_TYPE_TIMESTAMP {
					if condition.Operator == datapb.Condition_OPERATOR_LESS_THAN || condition.Operator == datapb.Condition_OPERATOR_LESS_THAN_OR_EQUAL {
						return Filters{}, errors.ErrDataSourceSpecHasInvalidTimeCondition
					}
				}
			}

			conditions, err := toConditions(f.Key.Type, f.Conditions)
			if err != nil {
				return Filters{}, err
			}

			typedFilter, ok := typedFilters[f.Key.Name]
			var dp uint64
			if f.Key.NumberDecimalPlaces != nil {
				dp = *f.Key.NumberDecimalPlaces
			}
			if !ok {
				typedFilters[f.Key.Name] = filter{
					propertyName:     f.Key.Name,
					propertyType:     f.Key.Type,
					numberOfDecimals: dp,
					conditions:       conditions,
				}
				continue
			}

			if typedFilter.propertyType != f.Key.Type {
				return Filters{}, errMismatchPropertyType(typedFilter.propertyName, typedFilter.propertyType, f.Key.Type)
			}

			typedFilter.conditions = append(typedFilter.conditions, conditions...)
		} else {
			if len(f.Conditions) < 1 {
				return Filters{}, errors.ErrInternalTimeDataSourceMissingConditions
			}

			if (f.Conditions[0].Operator == datapb.Condition_OPERATOR_LESS_THAN) || (f.Conditions[0].Operator == datapb.Condition_OPERATOR_LESS_THAN_OR_EQUAL) {
				return Filters{}, errors.ErrDataSourceSpecHasInvalidTimeCondition
			}

			// Currently VEGA network uses only one type of internal data source - time triggered
			// that uses the property name "vegaprotocol.builtin.timestamp"
			// https://github.com/vegaprotocol/specs/blob/master/protocol/0048-DSRI-data_source_internal.md#13-vega-time-changed
			conditions, err := toConditions(datapb.PropertyKey_TYPE_TIMESTAMP, f.Conditions)
			if err != nil {
				return Filters{}, err
			}
			typedFilters[f.Key.Name] = filter{
				propertyName: "vegaprotocol.builtin.timestamp",
				propertyType: datapb.PropertyKey_TYPE_TIMESTAMP,
				conditions:   []condition{conditions[0]},
			}
		}
	}

	return Filters{
		filters: typedFilters,
	}, nil
}

func (f Filters) Match(data map[string]string) (bool, error) {
	for propertyName, filter := range f.filters {
		dataValue, ok := data[propertyName]
		if !ok {
			return false, nil
		}

		for _, condition := range filter.conditions {
			if matched, err := condition(dataValue); !matched || err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

func (f Filters) EnsureBoundableProperty(property string, propType datapb.PropertyKey_Type) error {
	filter, ok := f.filters[property]
	if !ok {
		return fmt.Errorf("bound property \"%s\" not filtered by oracle spec", property)
	}

	if filter.propertyType != propType {
		return fmt.Errorf("bound type \"%v\" doesn't match filtered property type \"%s\"", propType, filter.propertyType)
	}

	return nil
}

var conditionConverters = map[datapb.PropertyKey_Type]func(*SpecCondition) (condition, error){
	datapb.PropertyKey_TYPE_INTEGER:   toIntegerCondition,
	datapb.PropertyKey_TYPE_DECIMAL:   toDecimalCondition,
	datapb.PropertyKey_TYPE_BOOLEAN:   toBooleanCondition,
	datapb.PropertyKey_TYPE_TIMESTAMP: toTimestampCondition,
	datapb.PropertyKey_TYPE_STRING:    toStringCondition,
}

func toConditions(typ datapb.PropertyKey_Type, cs []*SpecCondition) ([]condition, error) {
	converter, ok := conditionConverters[typ]
	if !ok {
		return nil, errUnsupportedPropertyType(typ)
	}

	conditions := make([]condition, 0, len(cs))
	for _, c := range cs {
		cond, err := converter(c)
		if err != nil {
			return nil, err
		}

		conditions = append(conditions, cond)
	}
	return conditions, nil
}

func toIntegerCondition(c *SpecCondition) (condition, error) {
	condValue, err := ToInteger(c.Value)
	if err != nil {
		return nil, err
	}

	matcher, ok := integerMatchers[c.Operator]
	if !ok {
		return nil, err
	}

	return func(dataValue string) (bool, error) {
		parsedDataValue, err := ToInteger(dataValue)
		if err != nil {
			return false, err
		}
		return matcher(parsedDataValue, condValue), nil
	}, nil
}

func ToInteger(value string) (*num.Int, error) {
	convertedValue, hasError := num.IntFromString(value, 10)
	if hasError {
		return nil, fmt.Errorf("value \"%s\" is not a valid integer", value)
	}
	return convertedValue, nil
}

var integerMatchers = map[datapb.Condition_Operator]func(*num.Int, *num.Int) bool{
	datapb.Condition_OPERATOR_EQUALS:                equalsInteger,
	datapb.Condition_OPERATOR_GREATER_THAN:          greaterThanInteger,
	datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL: greaterThanOrEqualInteger,
	datapb.Condition_OPERATOR_LESS_THAN:             lessThanInteger,
	datapb.Condition_OPERATOR_LESS_THAN_OR_EQUAL:    lessThanOrEqualInteger,
}

func equalsInteger(dataValue, condValue *num.Int) bool {
	return dataValue.EQ(condValue)
}

func greaterThanInteger(dataValue, condValue *num.Int) bool {
	return dataValue.GT(condValue)
}

func greaterThanOrEqualInteger(dataValue, condValue *num.Int) bool {
	return dataValue.GTE(condValue)
}

func lessThanInteger(dataValue, condValue *num.Int) bool {
	return dataValue.LT(condValue)
}

func lessThanOrEqualInteger(dataValue, condValue *num.Int) bool {
	return dataValue.LTE(condValue)
}

func toDecimalCondition(c *SpecCondition) (condition, error) {
	condValue, err := ToDecimal(c.Value)
	if err != nil {
		return nil, fmt.Errorf("error parsing decimal: %s", err.Error())
	}

	matcher, ok := decimalMatchers[c.Operator]
	if !ok {
		return nil, errUnsupportedOperatorForType(c.Operator, datapb.PropertyKey_TYPE_DECIMAL)
	}

	return func(dataValue string) (bool, error) {
		parsedDataValue, err := ToDecimal(dataValue)
		if err != nil {
			return false, fmt.Errorf("error parsing decimal: %s", err.Error())
		}
		return matcher(parsedDataValue, condValue), nil
	}, nil
}

func ToDecimal(value string) (num.Decimal, error) {
	return num.DecimalFromString(value)
}

var decimalMatchers = map[datapb.Condition_Operator]func(num.Decimal, num.Decimal) bool{
	datapb.Condition_OPERATOR_EQUALS:                equalsDecimal,
	datapb.Condition_OPERATOR_GREATER_THAN:          greaterThanDecimal,
	datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL: greaterThanOrEqualDecimal,
	datapb.Condition_OPERATOR_LESS_THAN:             lessThanDecimal,
	datapb.Condition_OPERATOR_LESS_THAN_OR_EQUAL:    lessThanOrEqualDecimal,
}

func equalsDecimal(dataValue, condValue num.Decimal) bool {
	return dataValue.Equal(condValue)
}

func greaterThanDecimal(dataValue, condValue num.Decimal) bool {
	return dataValue.GreaterThan(condValue)
}

func greaterThanOrEqualDecimal(dataValue, condValue num.Decimal) bool {
	return dataValue.GreaterThanOrEqual(condValue)
}

func lessThanDecimal(dataValue, condValue num.Decimal) bool {
	return dataValue.LessThan(condValue)
}

func lessThanOrEqualDecimal(dataValue, condValue num.Decimal) bool {
	return dataValue.LessThanOrEqual(condValue)
}

func toTimestampCondition(c *SpecCondition) (condition, error) {
	condValue, err := ToTimestamp(c.Value)
	if err != nil {
		return nil, err
	}

	matcher, ok := timestampMatchers[c.Operator]
	if !ok {
		return nil, errUnsupportedOperatorForType(c.Operator, datapb.PropertyKey_TYPE_TIMESTAMP)
	}

	return func(dataValue string) (bool, error) {
		parsedDataValue, err := ToTimestamp(dataValue)
		if err != nil {
			return false, err
		}
		return matcher(parsedDataValue, condValue), nil
	}, nil
}

func ToTimestamp(value string) (int64, error) {
	parsedValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return parsedValue, errors.ErrInvalidTimestamp
	}

	if parsedValue < 0 {
		return parsedValue, errors.ErrInvalidTimestamp
	}
	return parsedValue, nil
}

var timestampMatchers = map[datapb.Condition_Operator]func(int64, int64) bool{
	datapb.Condition_OPERATOR_EQUALS:                equalsTimestamp,
	datapb.Condition_OPERATOR_GREATER_THAN:          greaterThanTimestamp,
	datapb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL: greaterThanOrEqualTimestamp,
	datapb.Condition_OPERATOR_LESS_THAN:             lessThanTimestamp,
	datapb.Condition_OPERATOR_LESS_THAN_OR_EQUAL:    lessThanOrEqualTimestamp,
}

func equalsTimestamp(dataValue, condValue int64) bool {
	return dataValue == condValue
}

func greaterThanTimestamp(dataValue, condValue int64) bool {
	return dataValue > condValue
}

func greaterThanOrEqualTimestamp(dataValue, condValue int64) bool {
	return dataValue >= condValue
}

func lessThanTimestamp(dataValue, condValue int64) bool {
	return dataValue < condValue
}

func lessThanOrEqualTimestamp(dataValue, condValue int64) bool {
	return dataValue <= condValue
}

func toBooleanCondition(c *SpecCondition) (condition, error) {
	condValue, err := ToBoolean(c.Value)
	if err != nil {
		return nil, fmt.Errorf("error parsing boolean: %s", err.Error())
	}

	matcher, ok := booleanMatchers[c.Operator]
	if !ok {
		return nil, errUnsupportedOperatorForType(c.Operator, datapb.PropertyKey_TYPE_BOOLEAN)
	}

	return func(dataValue string) (bool, error) {
		parsedDataValue, err := ToBoolean(dataValue)
		if err != nil {
			return false, fmt.Errorf("error parsing boolean: %s", err.Error())
		}
		return matcher(parsedDataValue, condValue), nil
	}, nil
}

func ToBoolean(value string) (bool, error) {
	return strconv.ParseBool(value)
}

var booleanMatchers = map[datapb.Condition_Operator]func(bool, bool) bool{
	datapb.Condition_OPERATOR_EQUALS: equalsBoolean,
}

func equalsBoolean(dataValue, condValue bool) bool {
	return dataValue == condValue
}

func toStringCondition(c *SpecCondition) (condition, error) {
	matcher, ok := stringMatchers[c.Operator]
	if !ok {
		return nil, errUnsupportedOperatorForType(c.Operator, datapb.PropertyKey_TYPE_STRING)
	}

	return func(dataValue string) (bool, error) {
		return matcher(dataValue, c.Value), nil
	}, nil
}

var stringMatchers = map[datapb.Condition_Operator]func(string, string) bool{
	datapb.Condition_OPERATOR_EQUALS: equalsString,
}

func equalsString(dataValue, condValue string) bool {
	return dataValue == condValue
}

// errUnsupportedOperatorForType is returned when the property type does not
// support the specified operator.
func errUnsupportedOperatorForType(o datapb.Condition_Operator, t datapb.PropertyKey_Type) error {
	return fmt.Errorf("unsupported operator %s for type %s", o, t)
}

// errUnsupportedPropertyType is returned when the filter specifies an
// unsupported property key type.
func errUnsupportedPropertyType(typ datapb.PropertyKey_Type) error {
	return fmt.Errorf("property type %s", typ)
}

// errMismatchPropertyType is returned when a property is redeclared in
// conditions but with a different type.
func errMismatchPropertyType(prop string, first, newP datapb.PropertyKey_Type) error {
	return fmt.Errorf(
		"cannot redeclared property %s with different type, first %s then %s",
		prop, first, newP,
	)
}
