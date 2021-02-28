package oracles

import (
	"errors"
	"fmt"
	"strconv"

	oraclespb "code.vegaprotocol.io/vega/proto/oracles/v1"
)

var (
	// ErrMissingPubKeys is returned when the oraclespb.OracleSpec is missing
	// its public keys.
	ErrMissingPubKeys = errors.New("public keys are required")
	// ErrMissingPropertiesAndFilters is returned when the oraclespb.OracleSpec
	// has no expected properties nor filters. At least one of these should be
	// defined.
	ErrMissingFilters = errors.New("at least one filter is required")
	// ErrInvalidTimestamp is returned when the timestamp has a negative value
	// which may happen in case of unsigned integer overflow.
	ErrInvalidTimestamp = errors.New("invalid timestamp")
)

type OracleSpecID string

type OracleSpec struct {
	// id is a unique identifier for the OracleSpec
	id OracleSpecID
	// pubKeys list all the authorized public keys from where an OracleData can
	// come from.
	pubKeys map[string]struct{}
	// filters holds all the expected property keys with the conditions they
	// should match.
	filters map[string]*filter
	// Proto is the protobuf description of OracleSpec
	Proto oraclespb.OracleSpec
}

type filter struct {
	propertyName string
	propertyType oraclespb.PropertyKey_Type
	conditions   []condition
}

type condition func(string) (bool, error)

// NewOracleSpec build an OracleSpec from an oraclespb.OracleSpec in a form that
// suits the processing of the filters.
func NewOracleSpec(proto oraclespb.OracleSpec) (*OracleSpec, error) {
	if len(proto.PubKeys) == 0 {
		return nil, ErrMissingPubKeys
	}

	pubKeys := map[string]struct{}{}
	for _, pk := range proto.PubKeys {
		pubKeys[pk] = struct{}{}
	}

	if len(proto.Filters) == 0 {
		return nil, ErrMissingFilters
	}

	typedFilters := map[string]*filter{}
	for _, f := range proto.Filters {
		conditions, err := toConditions(f.Key.Type, f.Conditions)
		if err != nil {
			return nil, err
		}

		typedFilter, ok := typedFilters[f.Key.Name]

		if !ok {
			typedFilters[f.Key.Name] = &filter{
				propertyName: f.Key.Name,
				propertyType: f.Key.Type,
				conditions:   conditions,
			}
			continue
		}

		if typedFilter.propertyType != f.Key.Type {
			return nil, errMismatchPropertyType(typedFilter.propertyName, typedFilter.propertyType, f.Key.Type)
		}

		typedFilter.conditions = append(typedFilter.conditions, conditions...)
	}

	return &OracleSpec{
		id:      OracleSpecID(proto.Id),
		pubKeys: pubKeys,
		filters: typedFilters,
		Proto:   proto,
	}, nil
}

func (s OracleSpec) CanBindProperty(property string) bool {
	_, ok := s.filters[property]
	return ok
}

// MatchData indicates if a given OracleData matches the spec or not.
func (s *OracleSpec) MatchData(data OracleData) (bool, error) {
	if !containsRequiredPubKeys(data.PubKeys, s.pubKeys) {
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

// containsRequiredPubKeys verifies if all the public keys is the OracleData is
// matches the keys authorized by the OracleSpec.
func containsRequiredPubKeys(dataPKs []string, authPks map[string]struct{}) bool {
	for _, pk := range dataPKs {
		if _, ok := authPks[pk]; !ok {
			return false
		}
	}
	return true
}

var conditionConverters = map[oraclespb.PropertyKey_Type]func(*oraclespb.Condition) (condition, error){
	oraclespb.PropertyKey_TYPE_INTEGER:   toIntegerCondition,
	oraclespb.PropertyKey_TYPE_DECIMAL:   toDecimalCondition,
	oraclespb.PropertyKey_TYPE_BOOLEAN:   toBooleanCondition,
	oraclespb.PropertyKey_TYPE_TIMESTAMP: toTimestampCondition,
	oraclespb.PropertyKey_TYPE_STRING:    toStringCondition,
}

func toConditions(typ oraclespb.PropertyKey_Type, cs []*oraclespb.Condition) ([]condition, error) {
	converter, ok := conditionConverters[typ]
	if !ok {
		return nil, errUnsupportedPropertyType(typ)
	}

	conditions := []condition{}
	for _, c := range cs {
		cond, err := converter(c)
		if err != nil {
			return nil, err
		}

		conditions = append(conditions, cond)
	}
	return conditions, nil
}

func toIntegerCondition(c *oraclespb.Condition) (condition, error) {
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

func toInteger(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}

var integerMatchers = map[oraclespb.Condition_Operator]func(int64, int64) bool{
	oraclespb.Condition_OPERATOR_EQUALS:                equalsInteger,
	oraclespb.Condition_OPERATOR_GREATER_THAN:          greaterThanInteger,
	oraclespb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL: greaterThanOrEqualInteger,
	oraclespb.Condition_OPERATOR_LESS_THAN:             lessThanInteger,
	oraclespb.Condition_OPERATOR_LESS_THAN_OR_EQUAL:    lessThanOrEqualInteger,
}

func equalsInteger(dataValue, condValue int64) bool {
	return dataValue == condValue
}

func greaterThanInteger(dataValue, condValue int64) bool {
	return dataValue > condValue
}

func greaterThanOrEqualInteger(dataValue, condValue int64) bool {
	return dataValue >= condValue
}

func lessThanInteger(dataValue, condValue int64) bool {
	return dataValue < condValue
}

func lessThanOrEqualInteger(dataValue, condValue int64) bool {
	return dataValue <= condValue
}

func toDecimalCondition(c *oraclespb.Condition) (condition, error) {
	condValue, err := toDecimal(c.Value)
	if err != nil {
		return nil, err
	}

	matcher, ok := decimalMatchers[c.Operator]
	if !ok {
		return nil, errUnsupportedOperatorForType(c.Operator, oraclespb.PropertyKey_TYPE_DECIMAL)
	}

	return func(dataValue string) (bool, error) {
		parsedDataValue, err := toDecimal(dataValue)
		if err != nil {
			return false, err
		}
		return matcher(parsedDataValue, condValue), nil
	}, nil
}

func toDecimal(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

var decimalMatchers = map[oraclespb.Condition_Operator]func(float64, float64) bool{
	oraclespb.Condition_OPERATOR_EQUALS:                equalsDecimal,
	oraclespb.Condition_OPERATOR_GREATER_THAN:          greaterThanDecimal,
	oraclespb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL: greaterThanOrEqualDecimal,
	oraclespb.Condition_OPERATOR_LESS_THAN:             lessThanDecimal,
	oraclespb.Condition_OPERATOR_LESS_THAN_OR_EQUAL:    lessThanOrEqualDecimal,
}

func equalsDecimal(dataValue, condValue float64) bool {
	return dataValue == condValue
}

func greaterThanDecimal(dataValue, condValue float64) bool {
	return dataValue > condValue
}

func greaterThanOrEqualDecimal(dataValue, condValue float64) bool {
	return dataValue >= condValue
}

func lessThanDecimal(dataValue, condValue float64) bool {
	return dataValue < condValue
}

func lessThanOrEqualDecimal(dataValue, condValue float64) bool {
	return dataValue <= condValue
}

func toTimestampCondition(c *oraclespb.Condition) (condition, error) {
	condValue, err := toTimestamp(c.Value)
	if err != nil {
		return nil, err
	}

	matcher, ok := timestampMatchers[c.Operator]
	if !ok {
		return nil, errUnsupportedOperatorForType(c.Operator, oraclespb.PropertyKey_TYPE_TIMESTAMP)
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

var timestampMatchers = map[oraclespb.Condition_Operator]func(int64, int64) bool{
	oraclespb.Condition_OPERATOR_EQUALS:                equalsTimestamp,
	oraclespb.Condition_OPERATOR_GREATER_THAN:          greaterThanTimestamp,
	oraclespb.Condition_OPERATOR_GREATER_THAN_OR_EQUAL: greaterThanOrEqualTimestamp,
	oraclespb.Condition_OPERATOR_LESS_THAN:             lessThanTimestamp,
	oraclespb.Condition_OPERATOR_LESS_THAN_OR_EQUAL:    lessThanOrEqualTimestamp,
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

func toBooleanCondition(c *oraclespb.Condition) (condition, error) {
	condValue, err := toBoolean(c.Value)
	if err != nil {
		return nil, err
	}

	matcher, ok := booleanMatchers[c.Operator]
	if !ok {
		return nil, errUnsupportedOperatorForType(c.Operator, oraclespb.PropertyKey_TYPE_BOOLEAN)
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

var booleanMatchers = map[oraclespb.Condition_Operator]func(bool, bool) bool{
	oraclespb.Condition_OPERATOR_EQUALS: equalsBoolean,
}

func equalsBoolean(dataValue, condValue bool) bool {
	return dataValue == condValue
}

func toStringCondition(c *oraclespb.Condition) (condition, error) {
	matcher, ok := stringMatchers[c.Operator]
	if !ok {
		return nil, errUnsupportedOperatorForType(c.Operator, oraclespb.PropertyKey_TYPE_STRING)
	}

	return func(dataValue string) (bool, error) {
		return matcher(dataValue, c.Value), nil
	}, nil
}

var stringMatchers = map[oraclespb.Condition_Operator]func(string, string) bool{
	oraclespb.Condition_OPERATOR_EQUALS: equalsString,
}

func equalsString(dataValue, condValue string) bool {
	return dataValue == condValue
}

// errMismatchPropertyType is returned when a property is redeclared in
// conditions but with a different type.
func errMismatchPropertyType(prop string, first, new oraclespb.PropertyKey_Type) error {
	return fmt.Errorf(
		"cannot redeclared property %s with different type, first %s then %s",
		prop, first, new,
	)
}

// errUnsupportedOperatorForType is returned when the property type does not
// support the specified operator.
func errUnsupportedOperatorForType(o oraclespb.Condition_Operator, t oraclespb.PropertyKey_Type) error {
	return fmt.Errorf("unsupported operator %s for type %s", o, t)
}

// errUnsupportedPropertyType is returned when the filter specifies an
// unsupported property key type.
func errUnsupportedPropertyType(typ oraclespb.PropertyKey_Type) error {
	return fmt.Errorf("property type %s", typ)
}
