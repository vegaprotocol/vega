package execution

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestFeeSplitter(t *testing.T) {
	var (
		totalStake              uint64 = 100
		timeWindowStart                = time.Now()
		marketValueWindowLength        = 1 * time.Minute
	)

	tests := []struct {
		currentTime        time.Time
		tradedValue        uint64
		expectedValueProxy decimal.Decimal
	}{
		{
			currentTime:        timeWindowStart,
			expectedValueProxy: decimal.NewFromFloat(100.),
		},
		{
			tradedValue:        10,
			currentTime:        timeWindowStart.Add(10 * time.Second),
			expectedValueProxy: decimal.NewFromFloat(100.),
		},
		{
			tradedValue:        100,
			currentTime:        timeWindowStart.Add(30 * time.Second),
			expectedValueProxy: decimal.NewFromFloat(200.),
		},
		{
			tradedValue:        300,
			currentTime:        timeWindowStart.Add(3 * marketValueWindowLength),
			expectedValueProxy: decimal.NewFromFloat(300.),
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			fs := &FeeSplitter{}
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
