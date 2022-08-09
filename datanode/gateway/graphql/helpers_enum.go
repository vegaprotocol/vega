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

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
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

// convertIntervalToProto converts a GraphQL enum to a Proto enum.
func convertIntervalToProto(x Interval) (types.Interval, error) {
	switch x {
	case IntervalI1m:
		return types.Interval_INTERVAL_I1M, nil
	case IntervalI5m:
		return types.Interval_INTERVAL_I5M, nil
	case IntervalI15m:
		return types.Interval_INTERVAL_I15M, nil
	case IntervalI1h:
		return types.Interval_INTERVAL_I1H, nil
	case IntervalI6h:
		return types.Interval_INTERVAL_I6H, nil
	case IntervalI1d:
		return types.Interval_INTERVAL_I1D, nil
	default:
		err := fmt.Errorf("failed to convert Interval from GraphQL to Proto: %v", x)
		return types.Interval_INTERVAL_UNSPECIFIED, err
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

// convertIntervalFromProto converts a Proto enum to a GraphQL enum.
func convertIntervalFromProto(x types.Interval) (Interval, error) {
	switch x {
	case types.Interval_INTERVAL_I1M:
		return IntervalI1m, nil
	case types.Interval_INTERVAL_I5M:
		return IntervalI5m, nil
	case types.Interval_INTERVAL_I15M:
		return IntervalI15m, nil
	case types.Interval_INTERVAL_I1H:
		return IntervalI1h, nil
	case types.Interval_INTERVAL_I6H:
		return IntervalI6h, nil
	case types.Interval_INTERVAL_I1D:
		return IntervalI1d, nil
	default:
		err := fmt.Errorf("failed to convert Interval from Proto to GraphQL: %v", x)
		return IntervalI15m, err
	}
}

// convertOrderTypeToProto converts a GraphQL enum to a Proto enum.
func convertOrderTypeToProto(x OrderType) (types.Order_Type, error) {
	switch x {
	case OrderTypeLimit:
		return types.Order_TYPE_LIMIT, nil
	case OrderTypeMarket:
		return types.Order_TYPE_MARKET, nil
	case OrderTypeNetwork:
		return types.Order_TYPE_NETWORK, nil
	default:
		err := fmt.Errorf("failed to convert OrderType from GraphQL to Proto: %v", x)
		return types.Order_TYPE_UNSPECIFIED, err
	}
}

// convertOrderTypeFromProto converts a Proto enum to a GraphQL enum.
func convertOrderTypeFromProto(x types.Order_Type) (OrderType, error) {
	switch x {
	case types.Order_TYPE_LIMIT:
		return OrderTypeLimit, nil
	case types.Order_TYPE_MARKET:
		return OrderTypeMarket, nil
	case types.Order_TYPE_NETWORK:
		return OrderTypeNetwork, nil
	default:
		err := fmt.Errorf("failed to convert OrderType from Proto to GraphQL: %v", x)
		return OrderTypeLimit, err
	}
}

// convertMarketStateFromProto converts a Proto enum to a GraphQL enum.
func convertMarketTradingModeFromProto(ms types.Market_TradingMode) (MarketTradingMode, error) {
	switch ms {
	case types.Market_TRADING_MODE_OPENING_AUCTION:
		return MarketTradingModeOpeningAuction, nil
	case types.Market_TRADING_MODE_BATCH_AUCTION:
		return MarketTradingModeBatchAuction, nil
	case types.Market_TRADING_MODE_MONITORING_AUCTION:
		return MarketTradingModeMonitoringAuction, nil
	case types.Market_TRADING_MODE_CONTINUOUS:
		return MarketTradingModeContinuous, nil
	case types.Market_TRADING_MODE_NO_TRADING:
		return MarketTradingModeNoTrading, nil
	default:
		err := fmt.Errorf("failed to convert MarketTradingMode from Proto to GraphQL: %v", ms)
		return MarketTradingModeContinuous, err
	}
}

// convertMarketStateFromProto converts a Proto enum to a GraphQL enum.
func convertMarketStateFromProto(ms types.Market_State) (MarketState, error) {
	switch ms {
	case types.Market_STATE_PROPOSED:
		return MarketStateProposed, nil
	case types.Market_STATE_REJECTED:
		return MarketStateRejected, nil
	case types.Market_STATE_PENDING:
		return MarketStatePending, nil
	case types.Market_STATE_CANCELLED:
		return MarketStateCancelled, nil
	case types.Market_STATE_ACTIVE:
		return MarketStateActive, nil
	case types.Market_STATE_SUSPENDED:
		return MarketStateSuspended, nil
	case types.Market_STATE_CLOSED:
		return MarketStateClosed, nil
	case types.Market_STATE_TRADING_TERMINATED:
		return MarketStateTradingTerminated, nil
	case types.Market_STATE_SETTLED:
		return MarketStateSettled, nil
	default:
		err := fmt.Errorf("failed to convert MarketMode from Proto to GraphQL: %v", ms)
		return MarketStateActive, err
	}
}

// convertProposalStateToProto converts a GraphQL enum to a Proto enum.
func convertProposalTypeToProto(x ProposalType) v2.ListGovernanceDataRequest_Type {
	switch x {
	case ProposalTypeNewMarket:
		return v2.ListGovernanceDataRequest_TYPE_NEW_MARKET
	case ProposalTypeUpdateMarket:
		return v2.ListGovernanceDataRequest_TYPE_UPDATE_MARKET
	case ProposalTypeNetworkParameters:
		return v2.ListGovernanceDataRequest_TYPE_NETWORK_PARAMETERS
	case ProposalTypeNewAsset:
		return v2.ListGovernanceDataRequest_TYPE_NEW_ASSET
	case ProposalTypeUpdateAsset:
		return v2.ListGovernanceDataRequest_TYPE_UPDATE_ASSET
	case ProposalTypeNewFreeForm:
		return v2.ListGovernanceDataRequest_TYPE_NEW_FREE_FORM
	default:
		return v2.ListGovernanceDataRequest_TYPE_ALL
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
