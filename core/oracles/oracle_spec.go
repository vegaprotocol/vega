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

package oracles

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

var (
	// ErrMissingSigners is returned when the datapb.OracleSpec is missing
	// its signers.
	ErrMissingSigners = errors.New("signers are required")
	// ErrAtLeastOneFilterIsRequired is returned when the datapb.OracleSpec
	// has no expected properties nor filters. At least one of these should be
	// defined.
	ErrAtLeastOneFilterIsRequired = errors.New("at least one filter is required")
	// ErrInvalidTimestamp is returned when the timestamp has a negative value
	// which may happen in case of unsigned integer overflow.
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	// ErrMissingPropertyKey is returned when a property key is undefined.
	ErrMissingPropertyKey = errors.New("a property key is required")
	// ErrMissingPropertyName is returned when a property as no name.
	ErrMissingPropertyName = errors.New("a property name is required")
	// ErrInvalidPropertyKey is returned if validation finds a reserved Vega property key.
	ErrInvalidPropertyKey = errors.New("property key is reserved")
)

type OracleSpecID string

type OracleSpec struct {
	// id is a unique identifier for the OracleSpec
	id OracleSpecID

	// signers list all the authorized public keys from where an OracleData can
	// come from.
	signers map[string]struct{}

	// filters holds all the expected property keys with the conditions they
	// should match.
	filters map[string]*filter
	// OriginalSpec is the protobuf description of OracleSpec
	OriginalSpec *types.OracleSpec
}

type filter struct {
	propertyName     string
	propertyType     datapb.PropertyKey_Type
	numberOfDecimals uint64
	conditions       []condition
}

type condition func(string) (bool, error)

// NewOracleSpec builds an OracleSpec from a types.OracleSpec (currently uses one level below - types.ExternalDataSourceSpec) in a form that
// suits the processing of the filters.
// OracleSpec allows the existence of one and only one.
// Currently VEGA network utilises internal triggers in the oracle function path, even though
// the oracles are treated as external data sources.
// For this reason this function checks if the provided external type of data source definition
// contains a key name that indicates a builtin type of logic
// and if the given data source definition is an internal type of data source, for more context refer to
// https://github.com/vegaprotocol/specs/blob/master/protocol/0048-DSRI-data_source_internal.md#13-vega-time-changed
func NewOracleSpec(originalSpec types.ExternalDataSourceSpec) (*OracleSpec, error) {
	filtersFromSpec := []*types.DataSourceSpecFilter{}
	isExtType := false
	if originalSpec.Spec != nil {
		if originalSpec.Spec.Data != nil {
			filtersFromSpec = originalSpec.Spec.Data.GetFilters()
			isExtType = originalSpec.Spec.Data.IsExternal()
		}
	}

	if len(filtersFromSpec) == 0 {
		return nil, ErrAtLeastOneFilterIsRequired
	}

	builtInKey := false
	typedFilters := map[string]*filter{}
	for _, f := range filtersFromSpec {
		if isExtType {
			if types.DataSourceSpecPropertyKeyIsEmpty(f.Key) {
				return nil, ErrMissingPropertyKey
			}

			if len(f.Key.Name) == 0 {
				return nil, ErrMissingPropertyName
			}

			_, exist := typedFilters[f.Key.Name]
			if exist {
				return nil, types.ErrMultipleSameKeyNamesInFilterList
			}

			if strings.HasPrefix(f.Key.Name, "vegaprotocol.builtin") && f.Key.Type == datapb.PropertyKey_TYPE_TIMESTAMP {
				builtInKey = true
			}

			conditions, err := toConditions(f.Key.Type, f.Conditions)
			if err != nil {
				return nil, err
			}

			typedFilter, ok := typedFilters[f.Key.Name]
			var dp uint64
			if f.Key.NumberDecimalPlaces != nil {
				dp = *f.Key.NumberDecimalPlaces
			}
			if !ok {
				typedFilters[f.Key.Name] = &filter{
					propertyName:     f.Key.Name,
					propertyType:     f.Key.Type,
					numberOfDecimals: dp,
					conditions:       conditions,
				}
				continue
			}

			if typedFilter.propertyType != f.Key.Type {
				return nil, errMismatchPropertyType(typedFilter.propertyName, typedFilter.propertyType, f.Key.Type)
			}

			typedFilter.conditions = append(typedFilter.conditions, conditions...)
		} else {
			// Currently VEGA network uses only one type of internal data source - time triggered
			// that uses the property name "vegaprotocol.builtin.timestamp"
			// https://github.com/vegaprotocol/specs/blob/master/protocol/0048-DSRI-data_source_internal.md#13-vega-time-changed
			conditions, err := toConditions(datapb.PropertyKey_TYPE_TIMESTAMP, f.Conditions)
			if err != nil {
				return nil, err
			}
			typedFilters[f.String()] = &filter{
				propertyName: "vegaprotocol.builtin.timestamp",
				propertyType: datapb.PropertyKey_TYPE_TIMESTAMP,
				conditions:   conditions,
			}
		}
	}

	signers := map[string]struct{}{}
	if !builtInKey && isExtType {
		signersFromSpec := []*types.Signer{}
		if originalSpec.Spec != nil {
			if originalSpec.Spec.Data != nil {
				src := *originalSpec.Spec.Data

				signersFromSpec = src.GetSigners()
			}
		}

		if len(signersFromSpec) == 0 {
			return nil, ErrMissingSigners
		}

		for _, pk := range signersFromSpec {
			signers[pk.String()] = struct{}{}
		}
	}

	os := &OracleSpec{
		id:      OracleSpecID(originalSpec.Spec.ID),
		signers: signers,
		filters: typedFilters,
		OriginalSpec: &types.OracleSpec{
			ExternalDataSourceSpec: &originalSpec,
		},
	}

	return os, nil
}

func (s OracleSpec) EnsureBoundableProperty(property string, propType datapb.PropertyKey_Type) error {
	filter, ok := s.filters[property]
	if !ok {
		return fmt.Errorf("bound property \"%s\" not filtered by oracle spec", property)
	}

	if filter.propertyType != propType {
		return fmt.Errorf("bound type \"%v\" doesn't match filtered property type \"%s\"", propType, filter.propertyType)
	}

	return nil
}

func isInternalOracleData(data OracleData) bool {
	for k := range data.Data {
		if !strings.HasPrefix(k, BuiltinOraclePrefix) {
			return false
		}
	}

	return true
}

// MatchSigners tries to match the public keys from the provided OracleData object with the ones
// present in the Spec.
func (s *OracleSpec) MatchSigners(data OracleData) bool {
	return containsRequiredSigners(data.Signers, s.signers)
}

// MatchData indicates if a given OracleData matches the spec or not.
func (s *OracleSpec) MatchData(data OracleData) (bool, error) {
	// if the data contains the internal oracle timestamp key, and only that key,
	// then we do not need to verify the public keys as there will not be one

	if !isInternalOracleData(data) && !containsRequiredSigners(data.Signers, s.signers) {
		return false, nil
	}

	for propertyName, filter := range s.filters {
		dataValue, ok := data.Data[propertyName]
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

// containsRequiredSigners verifies if all the public keys in the OracleData
// are within the list of currently authorized by the OracleSpec.
func containsRequiredSigners(dataSigners []*types.Signer, authPks map[string]struct{}) bool {
	for _, signer := range dataSigners {
		if _, ok := authPks[signer.String()]; !ok {
			return false
		}
	}
	return true
}

var conditionConverters = map[datapb.PropertyKey_Type]func(*types.OracleSpecCondition) (condition, error){
	datapb.PropertyKey_TYPE_INTEGER:   toIntegerCondition,
	datapb.PropertyKey_TYPE_DECIMAL:   toDecimalCondition,
	datapb.PropertyKey_TYPE_BOOLEAN:   toBooleanCondition,
	datapb.PropertyKey_TYPE_TIMESTAMP: toTimestampCondition,
	datapb.PropertyKey_TYPE_STRING:    toStringCondition,
}

func toConditions(typ datapb.PropertyKey_Type, cs []*types.OracleSpecCondition) ([]condition, error) {
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

func toIntegerCondition(c *types.OracleSpecCondition) (condition, error) {
	condValue, err := toInteger(c.Value)
	if err != nil {
		return nil, err
	}

	matcher, ok := integerMatchers[c.Operator]
	if !ok {
		return nil, err
	}

	return func(dataValue string) (bool, error) {
		parsedDataValue, err := toInteger(dataValue)
		if err != nil {
			return false, err
		}
		return matcher(parsedDataValue, condValue), nil
	}, nil
}

func toInteger(value string) (*num.Int, error) {
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

func toDecimalCondition(c *types.OracleSpecCondition) (condition, error) {
	condValue, err := toDecimal(c.Value)
	if err != nil {
		return nil, err
	}

	matcher, ok := decimalMatchers[c.Operator]
	if !ok {
		return nil, errUnsupportedOperatorForType(c.Operator, datapb.PropertyKey_TYPE_DECIMAL)
	}

	return func(dataValue string) (bool, error) {
		parsedDataValue, err := toDecimal(dataValue)
		if err != nil {
			return false, err
		}
		return matcher(parsedDataValue, condValue), nil
	}, nil
}

func toDecimal(value string) (num.Decimal, error) {
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

func toTimestampCondition(c *types.OracleSpecCondition) (condition, error) {
	condValue, err := toTimestamp(c.Value)
	if err != nil {
		return nil, err
	}

	matcher, ok := timestampMatchers[c.Operator]
	if !ok {
		return nil, errUnsupportedOperatorForType(c.Operator, datapb.PropertyKey_TYPE_TIMESTAMP)
	}

	return func(dataValue string) (bool, error) {
		parsedDataValue, err := toTimestamp(dataValue)
		if err != nil {
			return false, err
		}
		return matcher(parsedDataValue, condValue), nil
	}, nil
}

func toTimestamp(value string) (int64, error) {
	parsedValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return parsedValue, err
	}

	if parsedValue < 0 {
		return parsedValue, ErrInvalidTimestamp
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

func toBooleanCondition(c *types.OracleSpecCondition) (condition, error) {
	condValue, err := toBoolean(c.Value)
	if err != nil {
		return nil, err
	}

	matcher, ok := booleanMatchers[c.Operator]
	if !ok {
		return nil, errUnsupportedOperatorForType(c.Operator, datapb.PropertyKey_TYPE_BOOLEAN)
	}

	return func(dataValue string) (bool, error) {
		parsedDataValue, err := toBoolean(dataValue)
		if err != nil {
			return false, err
		}
		return matcher(parsedDataValue, condValue), nil
	}, nil
}

func toBoolean(value string) (bool, error) {
	return strconv.ParseBool(value)
}

var booleanMatchers = map[datapb.Condition_Operator]func(bool, bool) bool{
	datapb.Condition_OPERATOR_EQUALS: equalsBoolean,
}

func equalsBoolean(dataValue, condValue bool) bool {
	return dataValue == condValue
}

func toStringCondition(c *types.OracleSpecCondition) (condition, error) {
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

// errMismatchPropertyType is returned when a property is redeclared in
// conditions but with a different type.
func errMismatchPropertyType(prop string, first, newP datapb.PropertyKey_Type) error {
	return fmt.Errorf(
		"cannot redeclared property %s with different type, first %s then %s",
		prop, first, newP,
	)
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
