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

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"go.uber.org/zap"
)

// Data holds normalized data coming from an oracle.
type Data struct {
	Signers  []*Signer
	Data     map[string]string
	MetaData map[string]string
}

func (d Data) GetUint(name string) (*num.Uint, error) {
	value, ok := d.Data[name]
	if !ok {
		return nil, errPropertyNotFound(name)
	}
	val, fail := num.UintFromString(value, 10)
	if fail {
		return nil, errInvalidString(name, value)
	}
	return val, nil
}

// GetInteger converts the value associated to propertyName into an integer.
func (d Data) GetInteger(propertyName string) (*num.Int, error) {
	value, ok := d.Data[propertyName]
	if !ok {
		return num.IntZero(), errPropertyNotFound(propertyName)
	}
	return ToInteger(value)
}

// GetDecimal converts the value associated to propertyName into a decimal.
func (d Data) GetDecimal(propertyName string) (num.Decimal, error) {
	value, ok := d.Data[propertyName]
	if !ok {
		return num.DecimalZero(), errPropertyNotFound(propertyName)
	}
	return ToDecimal(value)
}

// GetBoolean converts the value associated to propertyName into a boolean.
func (d Data) GetBoolean(propertyName string) (bool, error) {
	value, ok := d.Data[propertyName]
	if !ok {
		return false, errPropertyNotFound(propertyName)
	}
	return ToBoolean(value)
}

// GetString returns the value associated to propertyName.
func (d Data) GetString(propertyName string) (string, error) {
	value, ok := d.Data[propertyName]
	if !ok {
		return "", errPropertyNotFound(propertyName)
	}
	return value, nil
}

// GetTimestamp converts the value associated to propertyName into a timestamp.
func (d Data) GetTimestamp(propertyName string) (int64, error) {
	value, ok := d.Data[propertyName]
	if !ok {
		return 0, errPropertyNotFound(propertyName)
	}
	return ToTimestamp(value)
}

// FromInternalOracle returns true if the oracle data has been emitted by an
// internal oracle.
func (d Data) FromInternalOracle() bool {
	return len(d.Signers) == 0
}

func (d Data) Debug() []zap.Field {
	keys := ""
	for _, key := range d.Signers {
		keys += key.String() + " "
	}

	fields := []zap.Field{
		logging.String("Signers", keys),
	}
	for property, value := range d.Data {
		fields = append(fields, logging.String(property, value))
	}
	return fields
}

// errPropertyNotFound is returned when the property is not present in the Data.
func errPropertyNotFound(propertyName string) error {
	return fmt.Errorf("property \"%s\" not found", propertyName)
}

func errInvalidString(name, val string) error {
	return fmt.Errorf("could not parse value '%s' for property '%s'", val, name)
}
