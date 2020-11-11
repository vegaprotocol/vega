package dispatch

import (
	"fmt"

	"code.vegaprotocol.io/vega/logging"
)

type Collateral interface {
	UpdateGovernanceAsset(id string) error
}

type Assets interface {
	IsEnabled(string) bool
}

func GovernanceAssetUpdate(
	log *logging.Logger,
	assets Assets,
	collateral Collateral,
) func(value string) error {
	return func(value string) error {
		if !assets.IsEnabled(value) {
			log.Debug("tried to push a governance update with an non-enabled asset",
				logging.String("asset-id", value))
			return fmt.Errorf("invalid asset %v", value)
		}

		if err := collateral.UpdateGovernanceAsset(value); err != nil {
			log.Debug("unable to update governance asset in collateral",
				logging.String("asset-id", value),
				logging.Error(err))
			return fmt.Errorf("unable to update governance asset in collateral %w", err)
		}
		return nil
	}
}
