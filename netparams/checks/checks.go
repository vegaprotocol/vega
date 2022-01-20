package checks

import (
	"errors"
	"fmt"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/logging"
)

type Collateral interface {
	AssetExists(asset string) bool
}

type Assets interface {
	IsEnabled(asset string) bool
}

func MarginScalingFactor() func(interface{}) error {
	return func(v interface{}) error {
		sf := v.(*types.ScalingFactors)
		if sf.SearchLevel >= sf.InitialMargin || sf.InitialMargin >= sf.CollateralRelease {
			return errors.New("invalid scaling factors (searchLevel < initialMargin < collateralRelease)")
		}
		return nil
	}
}

func RewardAssetUpdate(
	log *logging.Logger,
	assets Assets,
	collateral Collateral,
) func(value string) error {
	return func(value string) error {
		if !assets.IsEnabled(value) {
			log.Debug("tried to push a reward update with an non-enabled asset",
				logging.String("asset-id", value))
			return fmt.Errorf("invalid asset %v", value)
		}

		if !collateral.AssetExists(value) {
			log.Debug("unable to update reward asset in collateral",
				logging.String("asset-id", value))
			return fmt.Errorf("asset does not exists in collateral %v", value)
		}
		return nil
	}
}
