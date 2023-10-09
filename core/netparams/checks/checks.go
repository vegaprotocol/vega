// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
