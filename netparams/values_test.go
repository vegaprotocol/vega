package netparams_test

import (
	"errors"
	"testing"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/netparams"
	"github.com/stretchr/testify/assert"
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
	j := netparams.NewJSON(&types.PriceMonitoringParameters{}, netparams.JSONProtoValidator()).Mutable(true).MustUpdate(validPmJSONString)
	assert.NotNil(t, j)
	err := j.Validate(validPmJSONString)
	assert.NoError(t, err)

	err = j.Update(validPmJSONString)
	assert.NoError(t, err)

	pm := &types.PriceMonitoringParameters{}
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
	j = netparams.NewJSON(&types.PriceMonitoringParameters{}, netparams.JSONProtoValidator()).Mutable(true).MustUpdate(validPmJSONString)
	assert.NotNil(t, j)
	err = j.Validate(validPmJSONString)
	assert.NoError(t, err)

	err = j.Update(validPmJSONString)
	assert.NoError(t, err)

	pm = &types.PriceMonitoringParameters{}
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
	expectedErrorMsg = "invalid field Triggers.Horizon: value '0' must be greater than '0'"
	err = j.Validate(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	err = j.Update(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	// probability
	invalidPmJSONString = `{"triggers": [{"horizon": 60, "probability": "0", "auction_extension": 90},{"horizon": 120, "probability": "0.99", "auction_extension": 180}]}`
	expectedErrorMsg = "invalid field Triggers.Probability: value '0' must be strictly greater than '0'"
	err = j.Validate(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	err = j.Update(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	invalidPmJSONString = `{"triggers": [{"horizon": 60, "probability": "1", "auction_extension": 90},{"horizon": 120, "probability": "0.99", "auction_extension": 180}]}`
	expectedErrorMsg = "invalid field Triggers.Probability: value '1' must be strictly lower than '1'"
	err = j.Validate(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	err = j.Update(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)

	// auctionExtension
	invalidPmJSONString = `{"triggers": [{"horizon": 60, "probability": "0.95", "auction_extension": 0},{"horizon": 120, "probability": "0.99", "auction_extension": 180}]}`
	expectedErrorMsg = "invalid field Triggers.AuctionExtension: value '0' must be greater than '0'"
	err = j.Validate(invalidPmJSONString)
	assert.EqualError(t, err, expectedErrorMsg)
}
