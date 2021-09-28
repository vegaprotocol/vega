package dispatch

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/logging"
)

type Assets interface {
	IsEnabled(string) bool
}

func RewardAssetUpdate(
	log *logging.Logger,
	assets Assets,
) func(context.Context, string) error {
	return func(ctx context.Context, value string) error {
		if !assets.IsEnabled(value) {
			log.Debug("tried to push a reward update with an non-enabled asset",
				logging.String("asset-id", value))
			return fmt.Errorf("invalid asset %v", value)
		}

		return nil
	}
}
