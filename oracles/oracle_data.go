package oracles

import (
	"fmt"

	"code.vegaprotocol.io/vega/logging"

	"go.uber.org/zap"
)

// OracleData holds normalized data coming from an oracle.
type OracleData struct {
	PubKeys []string
	Data    map[string]string
}

// GetInteger converts the value associated to propertyName into an integer.
func (d OracleData) GetInteger(propertyName string) (int64, error) {
	value, ok := d.Data[propertyName]
	if !ok {
		return 0, errPropertyNotFound(propertyName)
	}
	return toInteger(value)
}

// GetDecimal converts the value associated to propertyName into a decimal.
func (d OracleData) GetDecimal(propertyName string) (float64, error) {
	value, ok := d.Data[propertyName]
	if !ok {
		return 0, errPropertyNotFound(propertyName)
	}
	return toDecimal(value)
}

// GetBoolean converts the value associated to propertyName into a boolean.
func (d OracleData) GetBoolean(propertyName string) (bool, error) {
	value, ok := d.Data[propertyName]
	if !ok {
		return false, errPropertyNotFound(propertyName)
	}
	return toBoolean(value)
}

// GetString returns the value associated to propertyName.
func (d OracleData) GetString(propertyName string) (string, error) {
	value, ok := d.Data[propertyName]
	if !ok {
		return "", errPropertyNotFound(propertyName)
	}
	return value, nil
}

// GetTimestamp converts the value associated to propertyName into a timestamp.
func (d OracleData) GetTimestamp(propertyName string) (int64, error) {
	value, ok := d.Data[propertyName]
	if !ok {
		return 0, errPropertyNotFound(propertyName)
	}
	return toTimestamp(value)
}

func (d OracleData) Debug() []zap.Field {
	keys := ""
	for _, key := range d.PubKeys {
		keys += key + " "
	}

	fields := []zap.Field{
		logging.String("PubKeys", keys),
	}
	for property, value := range d.Data {
		fields = append(fields, logging.String(property, value))
	}
	return fields
}

// errPropertyNotFound is returned when the property is not present in the Data
func errPropertyNotFound(propertyName string) error {
	return fmt.Errorf("property \"%s\" not found", propertyName)
}
