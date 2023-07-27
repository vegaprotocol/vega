package gql

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/datanode/vegatime"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
)

func handleCandleConnectionRequest(ctx context.Context, client TradingDataServiceClientV2, market *vega.Market, sinceRaw string, toRaw *string,
	interval vega.Interval, pagination *v2.Pagination,
) (*v2.CandleDataConnection, error) {
	since, err := vegatime.Parse(sinceRaw)
	if err != nil {
		return nil, err
	}

	to := time.Unix(0, 0)
	if toRaw != nil {
		to, err = vegatime.Parse(*toRaw)
		if err != nil {
			return nil, err
		}
	}

	var mkt string
	if market != nil {
		mkt = market.Id
	}

	candlesForMktReq := v2.ListCandleIntervalsRequest{MarketId: mkt}
	candlesForMktResp, err := client.ListCandleIntervals(ctx, &candlesForMktReq)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve candles for market %s: %w", mkt, err)
	}

	requestInterval, err := toV2IntervalString(interval)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve candles for market %s: %w", mkt, err)
	}

	candleID := ""

	for _, c4m := range candlesForMktResp.IntervalToCandleId {
		if c4m.Interval == requestInterval {
			candleID = c4m.CandleId
			break
		}
	}

	if candleID == "" {
		return nil, fmt.Errorf("could not find candle for market %s and interval %s", mkt, interval)
	}

	newestFirst := false
	if pagination == nil {
		pagination = &v2.Pagination{}
	}

	pagination.NewestFirst = &newestFirst

	req := v2.ListCandleDataRequest{
		CandleId:      candleID,
		FromTimestamp: since.UnixNano(),
		ToTimestamp:   to.UnixNano(),
		Pagination:    pagination,
	}
	resp, err := client.ListCandleData(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve candles for market %s: %w", mkt, err)
	}

	return resp.Candles, nil
}

func toV2IntervalString(interval vega.Interval) (string, error) {
	switch interval {
	case vega.Interval_INTERVAL_BLOCK:
		return "block", nil
	case vega.Interval_INTERVAL_I1M:
		return "1 minute", nil
	case vega.Interval_INTERVAL_I5M:
		return "5 minutes", nil
	case vega.Interval_INTERVAL_I15M:
		return "15 minutes", nil
	case vega.Interval_INTERVAL_I1H:
		return "1 hour", nil
	case vega.Interval_INTERVAL_I6H:
		return "6 hours", nil
	case vega.Interval_INTERVAL_I1D:
		return "1 day", nil
	default:
		return "", fmt.Errorf("interval not support:%s", interval)
	}
}

func handleWithdrawalsConnectionRequest(ctx context.Context, client TradingDataServiceClientV2, party *vega.Party,
	dateRange *v2.DateRange, pagination *v2.Pagination,
) (*v2.WithdrawalsConnection, error) {
	req := v2.ListWithdrawalsRequest{PartyId: party.Id, Pagination: pagination, DateRange: dateRange}
	resp, err := client.ListWithdrawals(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve withdrawals for party %s: %w", party.Id, err)
	}
	return resp.Withdrawals, nil
}

func handleDepositsConnectionRequest(ctx context.Context, client TradingDataServiceClientV2, party *vega.Party,
	dateRange *v2.DateRange, pagination *v2.Pagination,
) (*v2.DepositsConnection, error) {
	req := v2.ListDepositsRequest{PartyId: party.Id, Pagination: pagination, DateRange: dateRange}
	resp, err := client.ListDeposits(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve deposits for party %s: %w", party.Id, err)
	}
	return resp.Deposits, nil
}

func handleProposalsRequest(ctx context.Context, client TradingDataServiceClientV2, party *vega.Party, ref *string, inType *v2.ListGovernanceDataRequest_Type,
	inState *vega.Proposal_State, pagination *v2.Pagination,
) (*v2.GovernanceDataConnection, error) {
	var partyID *string

	if party != nil {
		partyID = &party.Id
	}

	req := v2.ListGovernanceDataRequest{
		ProposerPartyId:   partyID,
		ProposalReference: ref,
		ProposalType:      inType,
		ProposalState:     inState,
		Pagination:        pagination,
	}
	resp, err := client.ListGovernanceData(ctx, &req)
	if err != nil {
		return nil, err
	}
	return resp.Connection, nil
}

func handleDelegationConnectionRequest(ctx context.Context, client TradingDataServiceClientV2,
	partyID, nodeID, epochID *string, pagination *v2.Pagination,
) (*v2.DelegationsConnection, error) {
	req := v2.ListDelegationsRequest{
		PartyId:    partyID,
		NodeId:     nodeID,
		EpochId:    epochID,
		Pagination: pagination,
	}

	resp, err := client.ListDelegations(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve requested delegations: %w", err)
	}
	return resp.Delegations, nil
}
