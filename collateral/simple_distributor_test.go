package collateral

import (
	"context"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func Test_simpleDistributor_Run(t *testing.T) {
	t.Parallel()
	now := time.Now().Unix()
	marketID := uuid.NewV4().String()
	tests := []struct {
		name            string
		expectCollected num.Decimal
		collected       num.Decimal
		requests        []request
		want            []events.Event
	}{
		{
			name:            "emit loss soc events",
			expectCollected: decimal.RequireFromString("2100"),
			collected:       decimal.RequireFromString("1600"),
			requests: []request{
				{
					amount: decimal.RequireFromString("1066.6666666666665"),
					request: &types.Transfer{
						Owner: "trader1",
						Amount: &types.FinancialAmount{
							Amount: num.NewUint(1400),
							Asset:  "BTC",
						},
						Type:      types.TransferType_TRANSFER_TYPE_MTM_WIN,
						MinAmount: num.NewUint(0),
					},
				},
				{
					amount: decimal.RequireFromString("533.3333333333333"),
					request: &types.Transfer{
						Owner: "trader2",
						Amount: &types.FinancialAmount{
							Amount: num.NewUint(700),
							Asset:  "BTC",
						},
						Type:      types.TransferType_TRANSFER_TYPE_MTM_WIN,
						MinAmount: num.NewUint(0),
					},
				},
			},
			want: []events.Event{
				events.NewLossSocializationEvent(context.Background(), "trader1", marketID, decimalPtr("-334"), nil, now),
				events.NewLossSocializationEvent(context.Background(), "trader2", marketID, decimalPtr("-166"), nil, now),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s := &simpleDistributor{
				log:             logging.NewTestLogger(),
				marketID:        marketID,
				expectCollected: tt.expectCollected,
				collected:       tt.collected,
				requests:        tt.requests,
				ts:              now,
			}
			got := s.Run(context.Background())

			require.Equal(t, 2, len(got))
			require.Equal(t, tt.want, got)
		})
	}
}