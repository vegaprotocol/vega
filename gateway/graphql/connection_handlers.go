package gql

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/data-node/vegatime"
	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	types "code.vegaprotocol.io/protos/vega"
)

func handleCandleConnectionRequest(ctx context.Context, client TradingDataServiceClientV2, market *types.Market, sinceRaw string, toRaw *string,
	interval Interval, pagination *v2.Pagination) (*v2.CandleDataConnection, error) {
	pInterval, err := convertIntervalToProto(interval)
	if err != nil {
		return nil, fmt.Errorf("could not convert interval: %w", err)
	}

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

	candlesForMktReq := v2.GetCandlesForMarketRequest{MarketId: mkt}
	candlesForMktResp, err := client.GetCandlesForMarket(ctx, &candlesForMktReq)

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
		Interval:      pInterval,
		Pagination:    pagination,
	}
	resp, err := client.ListCandleData(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve candles for market %s: %w", mkt, err)
	}

	return resp.Candles, nil
}

func handleWithdrawalsConnectionRequest(ctx context.Context, client TradingDataServiceClientV2, party *types.Party,
	pagination *v2.Pagination) (*v2.WithdrawalsConnection, error) {
	req := v2.ListWithdrawalsRequest{PartyId: party.Id, Pagination: pagination}
	resp, err := client.ListWithdrawals(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve withdrawals for party %s: %w", party.Id, err)
	}
	return resp.Withdrawals, nil
}

func handleDepositsConnectionRequest(ctx context.Context, client TradingDataServiceClientV2, party *types.Party,
	pagination *v2.Pagination) (*v2.DepositsConnection, error) {
	req := v2.ListDepositsRequest{PartyId: party.Id, Pagination: pagination}
	resp, err := client.ListDeposits(ctx, &req)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve deposits for party %s: %w", party.Id, err)
	}
	return resp.Deposits, nil
}

func handleProposalsRequest(ctx context.Context, client TradingDataServiceClientV2, party *types.Party, ref *string, inType *ProposalType,
	inState *ProposalState, pagination *v2.Pagination) (*v2.GovernanceDataConnection, error) {

	var partyID *string
	var proposalState *types.Proposal_State
	var proposalType *v2.ListGovernanceDataRequest_Type

	if party != nil {
		partyID = &party.Id
	}

	if inType != nil {
		pType := inType.IntoProtoValue()
		proposalType = &pType
	}

	if inState != nil {
		state, err := inState.IntoProtoValue()
		if err != nil {
			return nil, err
		}
		proposalState = &state
	}

	req := v2.ListGovernanceDataRequest{
		ProposerPartyId:   partyID,
		ProposalReference: ref,
		ProposalType:      proposalType,
		ProposalState:     proposalState,
		Pagination:        pagination,
	}
	resp, err := client.ListGovernanceData(ctx, &req)
	if err != nil {
		return nil, err
	}
	return resp.Connection, nil
}
func handleDelegationConnectionRequest(ctx context.Context, client TradingDataServiceClientV2,
	partyID, nodeID, epochID *string, pagination *v2.Pagination) (*v2.DelegationsConnection, error) {
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
