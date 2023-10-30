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

//go:build qa
// +build qa

package statevar

import (
	"math/rand"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

// AddNoise is a function used in qa build to add noise to the state variables within their tolerance to instrument consensus seeking.
func (sv *StateVariable) AddNoise(kvb []*vegapb.KeyValueBundle) []*vegapb.KeyValueBundle {
	for _, kvt := range kvb {
		tol, _ := num.DecimalFromString(kvt.Tolerance)
		switch v := kvt.Value.Value.(type) {
		case *vega.StateVarValue_ScalarVal:
			random := rand.Float64() * tol.InexactFloat64() / 2.0
			if sv.log.GetLevel() <= logging.DebugLevel {
				sv.log.Debug("adding random noise", logging.String("key-name", kvt.Key), logging.Float64("randomness", random))
			}
			val, _ := num.DecimalFromString(v.ScalarVal.Value)
			val = val.Add(num.DecimalFromFloat(random))
			kvt.Value.Value = &vegapb.StateVarValue_ScalarVal{
				ScalarVal: &vegapb.ScalarValue{
					Value: val.String(),
				},
			}

		case *vega.StateVarValue_VectorVal:
			vec := make([]num.Decimal, 0, len(v.VectorVal.Value))
			for i, entry := range v.VectorVal.Value {
				random := rand.Float64() * tol.InexactFloat64() / 2.0
				if sv.log.GetLevel() <= logging.DebugLevel {
					sv.log.Debug("adding random noise", logging.String("key-name", kvt.Key), logging.Int("index", i), logging.Float64("randomness", random))
				}
				value, _ := num.DecimalFromString(entry)
				vec = append(vec, value.Add(num.DecimalFromFloat(random)))
			}
			vecAsString := make([]string, 0, len(vec))
			for _, v := range vec {
				vecAsString = append(vecAsString, v.String())
			}
			kvt.Value.Value = &vegapb.StateVarValue_VectorVal{
				VectorVal: &vegapb.VectorValue{
					Value: vecAsString,
				},
			}
		}
	}
	return kvb
}
