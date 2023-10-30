// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package common

import (
	"fmt"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"go.uber.org/zap"
)

// Data holds normalized data coming from an oracle.
type Data struct {
	// EthKey is currently just the spec id, which is suboptimal as multiple specs calling the same
	// contracts at the same time will duplicate data. This is a temporary solution until we have
	// a better method of making a key of (contract address, args, block height/time + previous height/time)
	// 'previous' being required so that receivers can check if the their trigger would have fired.
	EthKey   string
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

// GetDataTimestampNano gets the eth block time (or vega time) associated with the oracle data.
func (d Data) GetDataTimestampNano() (int64, error) {
	if ebt, ok := d.MetaData["eth-block-time"]; ok {
		// add price point with "eth-block-time" as time
		pt, err := strconv.ParseInt(ebt, 10, 64)
		if err != nil {
			return 0, err
		}
		return time.Unix(pt, 0).UnixNano(), nil
	}
	// open oracle timestamp
	if oot, ok := d.MetaData["open-oracle-timestamp"]; ok {
		pt, err := strconv.ParseInt(oot, 10, 64)
		if err != nil {
			return 0, err
		}
		return time.Unix(pt, 0).UnixNano(), nil
	}
	// fall back to vega time
	if vt, ok := d.MetaData["vega-time"]; ok {
		t, err := strconv.ParseInt(vt, 10, 64)
		if err != nil {
			return 0, err
		}
		return time.Unix(t, 0).UnixNano(), nil
	}
	return 0, fmt.Errorf("data has no timestamp data")
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
