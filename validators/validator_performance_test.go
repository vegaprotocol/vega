package validators_test

import (
	"context"
	"encoding/hex"
	"testing"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/validators"
	"github.com/stretchr/testify/require"
)

var (
	tmkey1 = "uBr9FP/M/QyVtOa3j18+hjksXra7qxCa7e25/FVW5c0="
	tmkey4 = "g2QGqXnfkNmmVO4QSw/wu5kpk8rbaY+I4qYNzk7QJTc="

	address1, _ = hex.DecodeString("91484AD0B6343D73690F1D36A80EF92B67622C47")
	address2, _ = hex.DecodeString("3619F6EC431527F02457875B7355041ADBB54772")
	address3, _ = hex.DecodeString("13FA0B679D6064772567C7A6050B42CCA1C7C8CD")
)

func TestValidatorPerformanceNoPerformance(t *testing.T) {
	vp := validators.NewValidatorPerformance(logging.NewTestLogger())
	require.Equal(t, num.DecimalFromFloat(0.05), vp.ValidatorPerformanceScore("some name", 1, 10))
}

func TestElectedExpectationWithVotingPower(t *testing.T) {
	vp := validators.NewValidatorPerformance(logging.NewTestLogger())
	vp.BeginBlock(context.Background(), hex.EncodeToString(address1))
	vp.BeginBlock(context.Background(), hex.EncodeToString(address2))
	vp.BeginBlock(context.Background(), hex.EncodeToString(address1))
	vp.BeginBlock(context.Background(), hex.EncodeToString(address3))
	vp.BeginBlock(context.Background(), hex.EncodeToString(address2))

	// validator 1 proposed 2 times, out of 5 (i.e. 40%), they only got 20% voting power so they score should be capped at 1
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey1, 10, 50).String())

	// validator 1 proposed 2 times, out of 5 (i.e. 40%), they only got 40% voting power so they score should be 1
	require.Equal(t, "1", vp.ValidatorPerformanceScore(tmkey1, 20, 50).String())

	// validator 1 proposed 2 times, out of 5 (i.e. 40%), they got 60% voting power so they score should be 0.66667
	require.Equal(t, "0.6666666666666667", vp.ValidatorPerformanceScore(tmkey1, 30, 50).String())

	// validator 4 never proposed
	require.Equal(t, "0.05", vp.ValidatorPerformanceScore(tmkey4, 10, 50).String())
}
