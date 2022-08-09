// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gql

import (
	"fmt"

	types "code.vegaprotocol.io/vega/protos/vega"
)

func convertLiquidityProvisionStatusFromProto(x types.LiquidityProvision_Status) (LiquidityProvisionStatus, error) {
	switch x {
	case types.LiquidityProvision_STATUS_ACTIVE:
		return LiquidityProvisionStatusActive, nil
	case types.LiquidityProvision_STATUS_STOPPED:
		return LiquidityProvisionStatusStopped, nil
	case types.LiquidityProvision_STATUS_CANCELLED:
		return LiquidityProvisionStatusCancelled, nil
	case types.LiquidityProvision_STATUS_REJECTED:
		return LiquidityProvisionStatusRejected, nil
	case types.LiquidityProvision_STATUS_UNDEPLOYED:
		return LiquidityProvisionStatusUndeployed, nil
	case types.LiquidityProvision_STATUS_PENDING:
		return LiquidityProvisionStatusPending, nil
	default:
		err := fmt.Errorf("failed to convert LiquidityProvisionStatus from GraphQL to Proto: %v", x)
		return LiquidityProvisionStatusActive, err
	}
}

func convertDataNodeIntervalToProto(interval string) (types.Interval, error) {
	switch interval {
	case "1 minute":
		return types.Interval_INTERVAL_I1M, nil
	case "5 minutes":
		return types.Interval_INTERVAL_I5M, nil
	case "15 minutes":
		return types.Interval_INTERVAL_I15M, nil
	case "1 hour":
		return types.Interval_INTERVAL_I1H, nil
	case "6 hours":
		return types.Interval_INTERVAL_I6H, nil
	case "1 day":
		return types.Interval_INTERVAL_I1D, nil
	default:
		err := fmt.Errorf("failed to convert Interval from GraphQL to Proto: %v", interval)
		return types.Interval_INTERVAL_UNSPECIFIED, err
	}
}

// convertTradeTypeFromProto converts a Proto enum to a GraphQL enum.
func convertTradeTypeFromProto(x types.Trade_Type) (TradeType, error) {
	switch x {
	case types.Trade_TYPE_DEFAULT:
		return TradeTypeDefault, nil
	case types.Trade_TYPE_NETWORK_CLOSE_OUT_BAD:
		return TradeTypeNetworkCloseOutBad, nil
	case types.Trade_TYPE_NETWORK_CLOSE_OUT_GOOD:
		return TradeTypeNetworkCloseOutGood, nil
	default:
		err := fmt.Errorf("failed to convert TradeType from Proto to GraphQL: %v", x)
		return TradeTypeDefault, err
	}
}
