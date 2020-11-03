package checks

import (
	"fmt"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
)

type Collateral interface {
	AssetExists(asset string) bool
}

type Assets interface {
	IsEnabled(asset string) bool
}

func GovernanceAssetUpdate(
	log *logging.Logger,
	assets Assets,
	collateral Collateral,
) netparams.StringRule {
	return func(value string) error {
		fmt.Printf("CHECK ASSET EXISTS YOOOO\n\n\n")
		if !assets.IsEnabled(value) {
			log.Error("tried to push a governance update with an non-enabled asset",
				logging.String("asset-id", value))
			return fmt.Errorf("invalid asset %v", value)
		}

		if !collateral.AssetExists(value) {
			log.Error("unable to update governance asset in collateral",
				logging.String("asset-id", value))
			return fmt.Errorf("asset does not exists in collateral %v", value)
		}
		return nil
	}
}
