//go:build qa
// +build qa

package statevar

import (
	"math/rand"

	"code.vegaprotocol.io/protos/vega"
	vegapb "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types/num"
)

// AddNoise is a function used in qa build to add noise to the state variables within their tolerance to instrument consensus seeking.
func (sv *StateVariable) AddNoise(kvb []*vegapb.KeyValueBundle) []*vegapb.KeyValueBundle {
	for _, kvt := range kvb {
		tol, _ := num.DecimalFromString(kvt.Tolerance)
		switch v := kvt.Value.Value.(type) {
		case *vega.StateVarValue_ScalarVal:
			random := rand.Float64() * tol.InexactFloat64() / 2.0
			sv.log.Info("adding random noise", logging.String("key-name", kvt.Key), logging.Float64("randomness", random))
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
				sv.log.Info("adding random noise", logging.String("key-name", kvt.Key), logging.Int("index", i), logging.Float64("randomness", random))
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
