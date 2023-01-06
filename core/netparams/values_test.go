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

package netparams_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type A struct {
	S string
	I int
}

func (a *A) Reset() { *a = A{} }

type B struct {
	F  float32
	SS []string
}

func (b *B) Reset() { *b = B{} }

func TestNegativeUint(t *testing.T) {
	require.Error(t, errors.New("invalid uint"), netparams.NewUint().Mutable(true).Update("-30"))
	require.Error(t, errors.New("invalid uint"), netparams.NewUint().Mutable(true).Update("- 30"))
	require.Error(t, errors.New("invalid uint"), netparams.NewUint().Mutable(true).Update("    -     3 0"))
	require.NoError(t, netparams.NewUint().Mutable(true).Update("30"))
}

func TestJSONValues(t *testing.T) {
	validator := func(v interface{}) error {
		a, ok := v.(*A)
		if !ok {
			return errors.New("invalid type")
		}

		if len(a.S) <= 0 {
			return errors.New("empty string")
		}
		if a.I < 0 {
			return errors.New("I negative")
		}
		return nil
	}

	// happy case, all good
	j := netparams.NewJSON(&A{}, validator).Mutable(true).MustUpdate(`{"s": "notempty", "i": 42}`)
	assert.NotNil(t, j)
	err := j.Validate(`{"s": "notempty", "i": 84}`)
	assert.NoError(t, err)

	err = j.Update(`{"s": "notempty", "i": 84}`)
	assert.NoError(t, err)

	a := &A{}
	err = j.ToJSONStruct(a)
	assert.NoError(t, err)

	assert.Equal(t, a.I, 84)
	assert.Equal(t, a.S, "notempty")

	// errors cases now

	// invalid field
	err = j.Validate(`{"s": "notempty", "i": 84, "nope": 3.2}`)
	assert.EqualError(t, err, "unable to unmarshal value, json: unknown field \"nope\"")

	err = j.Update(`{"s": "notempty", "i": 84, "nope": 3.2}`)
	assert.EqualError(t, err, "unable to unmarshal value, json: unknown field \"nope\"")

	// invalid type
	b := &B{}
	err = j.ToJSONStruct(b)
	assert.EqualError(t, err, "incompatible type")

	// valid type, field validation failed
	err = j.Update(`{"s": "", "i": 84}`)
	assert.EqualError(t, err, "empty string")
}

func TestJSONVPriceMonitoringParameters(t *testing.T) {
	// happy case, populated parameters array
	validPmJSONString := `{"triggers": [{"horizon": 60, "probability": "0.95", "auction_extension": 90},{"horizon": 120, "probability": "0.99", "auction_extension": 180}]}`
	j := netparams.NewJSON(&vega.PriceMonitoringParameters{}, netparams.PriceMonitoringParametersValidation).Mutable(true).MustUpdate(validPmJSONString)
	assert.NotNil(t, j)
	err := j.Validate(validPmJSONString)
	assert.NoError(t, err)

	err = j.Update(validPmJSONString)
	assert.NoError(t, err)

	pm := &vega.PriceMonitoringParameters{}
	err = j.ToJSONStruct(pm)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(pm.Triggers))
	assert.Equal(t, int64(60), pm.Triggers[0].Horizon)
	assert.Equal(t, "0.95", pm.Triggers[0].Probability)
	assert.Equal(t, int64(90), pm.Triggers[0].AuctionExtension)
	assert.Equal(t, int64(120), pm.Triggers[1].Horizon)
	assert.Equal(t, "0.99", pm.Triggers[1].Probability)
	assert.Equal(t, int64(180), pm.Triggers[1].AuctionExtension)

	// happy case, empty parameters array
	validPmJSONString = `{"triggers": []}`
	j = netparams.NewJSON(&vega.PriceMonitoringParameters{}, netparams.PriceMonitoringParametersValidation).Mutable(true).MustUpdate(validPmJSONString)
	assert.NotNil(t, j)
	err = j.Validate(validPmJSONString)
	assert.NoError(t, err)

	err = j.Update(validPmJSONString)
	assert.NoError(t, err)

	pm = &vega.PriceMonitoringParameters{}
	err = j.ToJSONStruct(pm)
	assert.NoError(t, err)

	assert.Equal(t, 0, len(pm.Triggers))

	// errors cases now

	// invalid field
	invalidPmJSONString := `{"triggers": [{"horizon": 60, "probability": "0.95", "auction_extension": 90},{"horizon": 120, "probability": "0.99", "auction_extension": 180, "nope": "abc"}]}`
	expectedErrorMsg := "unable to unmarshal value, json: unknown field \"nope\""
	err = j.Validate(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	err = j.Update(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	// invalid value

	// horizon
	invalidPmJSONString = `{"triggers": [{"horizon": 0, "probability": "0.95", "auction_extension": 90},{"horizon": 120, "probability": "0.99", "auction_extension": 180}]}`
	expectedErrorMsg = "triggers.horizon must be greater than `0`, got `0`"
	err = j.Validate(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	err = j.Update(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	// probability
	invalidPmJSONString = `{"triggers": [{"horizon": 60, "probability": "0", "auction_extension": 90},{"horizon": 120, "probability": "0.99", "auction_extension": 180}]}`
	expectedErrorMsg = "triggers.probability must be greater than `0`, got `0`"
	err = j.Validate(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	err = j.Update(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	invalidPmJSONString = `{"triggers": [{"horizon": 60, "probability": "1", "auction_extension": 90},{"horizon": 120, "probability": "0.99", "auction_extension": 180}]}`
	expectedErrorMsg = "triggers.probability must be lower than `1`, got `1`"
	err = j.Validate(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	err = j.Update(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	// auctionExtension
	invalidPmJSONString = `{"triggers": [{"horizon": 60, "probability": "0.95", "auction_extension": 0},{"horizon": 120, "probability": "0.99", "auction_extension": 180}]}`
	expectedErrorMsg = "triggers.auction_extension must be greater than `0`, got `0`"
	err = j.Validate(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)
}

func TestDependent(t *testing.T) {
	u1 := netparams.NewUint(netparams.UintGTE(num.NewUint(1)), netparams.UintLTE(num.NewUint(1000))).Mutable(true).MustUpdate("500")
	u2 := netparams.NewUint(netparams.UintGTE(num.NewUint(1)), netparams.UintLTE(num.NewUint(1000))).Mutable(true).MustUpdate("100")
	// we want to enforce u1 >= 2x u2
	u1.AddRules(netparams.UintDependentGTE("u2", u2, num.MustDecimalFromString("2")))
	u2.AddRules(netparams.UintDependentLTE("u1", u1, num.MustDecimalFromString("0.5")))

	// try to update u1 to less than 2 * u2
	require.Equal(t, "expect >= 200 (u2 * 2) got 100", u1.Update("100").Error())
	require.NoError(t, u1.Update("200"))

	// try to update u2 to more than 0.5 u1
	require.Equal(t, "expect <= 100 (u1 * 0.5) got 101", u2.Update("101").Error())
	require.NoError(t, u2.Update("99"))
}
