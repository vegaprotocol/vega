package execution_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/execution"
	"code.vegaprotocol.io/data-node/types/num"

	"github.com/stretchr/testify/require"
)

func TestFeeSplitter(t *testing.T) {
	var (
		totalStake              = num.NewUint(100)
		timeWindowStart         = time.Now()
		marketValueWindowLength = 1 * time.Minute
	)

	tests := []struct {
		currentTime        time.Time
		tradedValue        *num.Uint
		expectedValueProxy num.Decimal
	}{
		{
			tradedValue:        num.Zero(),
			currentTime:        timeWindowStart,
			expectedValueProxy: num.DecimalFromFloat(100),
		},
		{
			tradedValue:        num.NewUint(10),
			currentTime:        timeWindowStart.Add(10 * time.Second),
			expectedValueProxy: num.DecimalFromFloat(100),
		},
		{
			tradedValue:        num.NewUint(100),
			currentTime:        timeWindowStart.Add(30 * time.Second),
			expectedValueProxy: num.DecimalFromFloat(200),
		},
		{
			tradedValue:        num.NewUint(300),
			currentTime:        timeWindowStart.Add(3 * marketValueWindowLength),
			expectedValueProxy: num.DecimalFromFloat(300),
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			fs := execution.NewFeeSplitter()
			fs.TimeWindowStart(timeWindowStart)
			require.NoError(t,
				fs.SetCurrentTime(test.currentTime),
			)
			fs.AddTradeValue(test.tradedValue)

			got := fs.MarketValueProxy(marketValueWindowLength, totalStake)
			require.True(t, test.expectedValueProxy.Equal(got))
		})
	}
}
