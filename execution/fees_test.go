package execution

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestFeeSplitter(t *testing.T) {
	var (
		totalStake              float64 = 100
		timeWindowStart                 = time.Now()
		marketValueWindowLength         = 1 * time.Minute
	)

	tests := []struct {
		currentTime        time.Time
		tradedValue        uint64
		expectedValueProxy float64
	}{
		{
			currentTime:        timeWindowStart,
			expectedValueProxy: 100,
		},
		{
			tradedValue:        10,
			currentTime:        timeWindowStart.Add(10 * time.Second),
			expectedValueProxy: 100,
		},
		{
			tradedValue:        100,
			currentTime:        timeWindowStart.Add(30 * time.Second),
			expectedValueProxy: 200,
		},
		{
			tradedValue:        300,
			currentTime:        timeWindowStart.Add(3 * marketValueWindowLength),
			expectedValueProxy: 300,
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
			require.Equal(t, test.expectedValueProxy, got)
		})
	}
}
