package types_test

import (
	"testing"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
	"github.com/stretchr/testify/require"
)

func TestPriceSettingsMapping(t *testing.T) {
	t1 := proto.PriceMonitoringTrigger{Horizon: 7200, Probability: 0.95, AuctionExtension: 300}
	t2 := proto.PriceMonitoringTrigger{Horizon: 3600, Probability: 0.99, AuctionExtension: 60}

	pSet := &proto.PriceMonitoringSettings{
		Parameters: &proto.PriceMonitoringParameters{
			Triggers: []*proto.PriceMonitoringTrigger{&t1, &t2},
		},
		UpdateFrequency: 600,
	}
	settings := types.PriceMonitoringSettingsFromProto(pSet)
	require.Equal(t, len(pSet.Parameters.Triggers), len(settings.Parameters.Triggers))
}
