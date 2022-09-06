package gql

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"code.vegaprotocol.io/vega/datanode/gateway"
	"code.vegaprotocol.io/vega/datanode/vegatime"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	types "code.vegaprotocol.io/vega/protos/vega"
)

func handleCandleConnectionRequest(ctx context.Context, client TradingDataServiceClientV2, market *vega.Market, sinceRaw string, toRaw *string,
	interval vega.Interval, pagination *v2.Pagination, log *logging.Logger,
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
	header := metadata.MD{}
	candlesForMktResp, err := client.ListCandleIntervals(ctx, &candlesForMktReq, grpc.Header(&header))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve candles for market %s: %w", mkt, err)
	}

	candleID := ""

	for _, c4m := range candlesForMktResp.IntervalToCandleId {
		if c4m.Interval == string(interval) {
			candleID = c4m.CandleId
			break
		}
	}

	if candleID == "" {
		return nil, fmt.Errorf("could not find candle for market %s and interval %s", mkt, interval)
	}

	req := v2.ListCandleDataRequest{
		CandleId:      candleID,
		FromTimestamp: since.Unix(),
		ToTimestamp:   to.Unix(),
		Interval:      interval,
		Pagination:    pagination,
	}

	header1 := metadata.MD{}
	resp, err := client.ListCandleData(ctx, &req, grpc.Header(&header1))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve candles for market %s: %w", mkt, err)
	}

	for k, v := range header1 {
		header[k] = v
	}

	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		log.Error("failed to add headers to context", logging.Error(err))
	}

	return resp.Candles, nil
}

func handleWithdrawalsConnectionRequest(ctx context.Context, client TradingDataServiceClientV2, party *types.Party,
	dateRange *v2.DateRange, pagination *v2.Pagination, log *logging.Logger,
) (*v2.WithdrawalsConnection, error) {
	req := v2.ListWithdrawalsRequest{PartyId: party.Id, Pagination: pagination, DateRange: dateRange}
	header := metadata.MD{}
	resp, err := client.ListWithdrawals(ctx, &req, grpc.Header(&header))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve withdrawals for party %s: %w", party.Id, err)
	}

	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		log.Error("failed to add headers to context", logging.Error(err))
	}

	return resp.Withdrawals, nil
}

func handleDepositsConnectionRequest(ctx context.Context, client TradingDataServiceClientV2, party *types.Party,
	dateRange *v2.DateRange, pagination *v2.Pagination, log *logging.Logger,
) (*v2.DepositsConnection, error) {
	req := v2.ListDepositsRequest{PartyId: party.Id, Pagination: pagination, DateRange: dateRange}
	header := metadata.MD{}
	resp, err := client.ListDeposits(ctx, &req, grpc.Header(&header))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve deposits for party %s: %w", party.Id, err)
	}

	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		log.Error("failed to add headers to context", logging.Error(err))
	}

	return resp.Deposits, nil
}

func handleProposalsRequest(ctx context.Context, client TradingDataServiceClientV2, party *types.Party, ref *string, inType *v2.ListGovernanceDataRequest_Type,
	inState *vega.Proposal_State, pagination *v2.Pagination, log *logging.Logger,
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
	header := metadata.MD{}
	resp, err := client.ListGovernanceData(ctx, &req, grpc.Header(&header))
	if err != nil {
		return nil, err
	}

	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		log.Error("failed to add headers to context", logging.Error(err))
	}

	return resp.Connection, nil
}

func handleDelegationConnectionRequest(ctx context.Context, client TradingDataServiceClientV2,
	partyID, nodeID, epochID *string, pagination *v2.Pagination, log *logging.Logger,
) (*v2.DelegationsConnection, error) {
	req := v2.ListDelegationsRequest{
		PartyId:    partyID,
		NodeId:     nodeID,
		EpochId:    epochID,
		Pagination: pagination,
	}

	header := metadata.MD{}
	resp, err := client.ListDelegations(ctx, &req, grpc.Header(&header))
	if err != nil {
		return nil, fmt.Errorf("could not retrieve requested delegations: %w", err)
	}

	if err = gateway.AddMDHeadersToContext(ctx, header); err != nil {
		log.Error("failed to add headers to context", logging.Error(err))
	}

	return resp.Delegations, nil
}
