// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities

import (
	"code.vegaprotocol.io/protos/vega"
	v1 "code.vegaprotocol.io/protos/vega/oracles/v1"
	"google.golang.org/protobuf/encoding/protojson"
)

type TradableInstrument struct {
	*vega.TradableInstrument
}

func (ti TradableInstrument) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(ti)
}

func (ti *TradableInstrument) UnmarshalJSON(data []byte) error {
	ti.TradableInstrument = &vega.TradableInstrument{}
	return protojson.Unmarshal(data, ti)
}

func (ti TradableInstrument) ToProto() *vega.TradableInstrument {
	return ti.TradableInstrument
}

func filtersFromProto(filters []*v1.Filter) []Filter {
	if len(filters) == 0 {
		return nil
	}

	results := make([]Filter, 0, len(filters))
	for _, filter := range filters {
		conditions := make([]Condition, 0, len(filter.Conditions))

		for _, condition := range filter.Conditions {
			conditions = append(conditions, Condition{
				Operator: condition.Operator,
				Value:    condition.Value,
			})
		}

		results = append(results, Filter{
			Key: PropertyKey{
				Name: filter.Key.Name,
				Type: filter.Key.Type,
			},
			Conditions: conditions,
		})
	}

	return results
}

func filtersToProto(filters []Filter) []*v1.Filter {
	if len(filters) == 0 {
		return nil
	}

	results := make([]*v1.Filter, 0, len(filters))
	for _, filter := range filters {
		conditions := make([]*v1.Condition, 0, len(filter.Conditions))
		for _, condition := range filter.Conditions {
			conditions = append(conditions, &v1.Condition{
				Operator: condition.Operator,
				Value:    condition.Value,
			})
		}

		results = append(results, &v1.Filter{
			Key: &v1.PropertyKey{
				Name: filter.Key.Name,
				Type: filter.Key.Type,
			},
			Conditions: conditions,
		})
	}

	return results
}

type Filter struct {
	Key        PropertyKey `json:"key"`
	Conditions []Condition `json:"conditions"`
}

type PropertyKey struct {
	Name string `json:"name"`
	Type v1.PropertyKey_Type
}

type Condition struct {
	Operator v1.Condition_Operator
	Value    string
}
