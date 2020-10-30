package netparams_test

import (
	"errors"
	"testing"

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
