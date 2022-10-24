// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package checks

import (
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/protos/vega"
)

type Collateral interface {
	AssetExists(asset string) bool
}

type Assets interface {
	IsEnabled(asset string) bool
}

func SpamPoWHashFunction(supportedFunctions []string) func(string) error {
	return func(name string) error {
		for _, v := range supportedFunctions {
			if v == name {
				return nil
			}
		}
		return errors.New("Spam Proof of Work hash function must be SHA3")
	}
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

func MarginScalingFactorRange(min, max num.Decimal) func(interface{}) error {
	return func(v interface{}) error {
		sf := v.(*types.ScalingFactors)
		if sf.SearchLevel < min.InexactFloat64() || sf.CollateralRelease > max.InexactFloat64() {
			return errors.New("invalid scaling factors (" + min.String() + "< searchLevel < initialMargin < collateralRelease <=" + max.String() + ")")
		}
		return nil
	}
}

func PriceMonitoringParametersAuctionExtension(min, max time.Duration) func(interface{}) error {
	return func(v interface{}) error {
		pmp := v.(*types.PriceMonitoringParameters)
		for _, pmt := range pmp.Triggers {
			if time.Duration(pmt.AuctionExtension*int64(time.Second)) < min || time.Duration(pmt.AuctionExtension*int64(time.Second)) > max {
				return errors.New("invalid AuctionExtension: must be between " + min.String() + " and " + max.String())
			}
		}
		return nil
	}
}

func PriceMonitoringParametersHorizon(min, max time.Duration) func(interface{}) error {
	return func(v interface{}) error {
		pmp := v.(*types.PriceMonitoringParameters)
		for _, pmt := range pmp.Triggers {
			if time.Duration(pmt.Horizon*int64(time.Second)) < min || time.Duration(pmt.Horizon*int64(time.Second)) > max {
				return errors.New("invalid Horizon: must be between " + min.String() + " and " + max.String())
			}
		}
		return nil
	}
}

func PriceMonitoringParametersProbability(min, max num.Decimal) func(interface{}) error {
	return func(v interface{}) error {
		pmp := v.(*types.PriceMonitoringParameters)
		for _, pmt := range pmp.Triggers {
			p, e := num.DecimalFromString(pmt.Probability)
			if e != nil {
				return e
			}
			if p.LessThan(min) || p.GreaterThanOrEqual(max) {
				return errors.New("invalid Probability: must be " + min.String() + " <= x < " + max.String())
			}
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
