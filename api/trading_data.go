// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package api

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"

	"code.vegaprotocol.io/data-node/candlesv2"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/service"
	"code.vegaprotocol.io/data-node/vegatime"
	pbtypes "code.vegaprotocol.io/protos/vega"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/types/num"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/metrics"
	"code.vegaprotocol.io/data-node/sqlstore"
	protoapi "code.vegaprotocol.io/protos/data-node/api/v1"
	"code.vegaprotocol.io/protos/vega"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"google.golang.org/grpc/codes"
)

type tradingDataService struct {
	protoapi.UnimplementedTradingDataServiceServer
	log          *logging.Logger
	Config       Config
	eventService EventService
	// *tradingDataService
	orderService              *service.Order
	tradeService              *service.Trade
	assetService              *service.Asset
	accountService            *service.Account
	marketDataService         *service.MarketData
	rewardService             *service.Reward
	marketService             *service.Markets
	delegationService         *service.Delegation
	epochService              *service.Epoch
	depositService            *service.Deposit
	withdrawalService         *service.Withdrawal
	governanceService         *service.Governance
	riskFactorService         *service.RiskFactor
	riskService               *service.Risk
	networkParameterService   *service.NetworkParameter
	blockService              *service.Block
	checkpointService         *service.Checkpoint
	partyService              *service.Party
	candleService             *candlesv2.Svc
	oracleSpecService         *service.OracleSpec
	oracleDataService         *service.OracleData
	liquidityProvisionService *service.LiquidityProvision
	positionService           *service.Position
	transferService           *service.Transfer
	stakeLinkingService       *service.StakeLinking
	notaryService             *service.Notary
	keyRotationService        *service.KeyRotations
	nodeService               *service.Node
	marketDepthService        *service.MarketDepth
	ledgerService             *service.Ledger
}

var defaultEntityPagination = entities.OffsetPagination{
	Skip:       0,
	Limit:      50,
	Descending: true,
}

/****************************** Ledger **************************************/
// TransferResponsesSubscribe opens a subscription to transfer response data provided by the transfer response service.
func (t *tradingDataService) TransferResponsesSubscribe(
	_ *protoapi.TransferResponsesSubscribeRequest, srv protoapi.TradingDataService_TransferResponsesSubscribeServer,
) error {
	// Wrap context from the request into cancellable. We can close internal chan in error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	transferResponsesChan, ref := t.ledgerService.Observe(ctx, t.Config.StreamRetries)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("TransferResponses subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observe(ctx, t.log, "TransferResponse", transferResponsesChan, ref, func(tr *vega.TransferResponse) error {
		return srv.Send(&protoapi.TransferResponsesSubscribeResponse{
			Response: tr,
		})
	})
}

/****************************** Market Depth **************************************/
// MarketDepth provides the order book for a given market, and also returns the most recent trade
// for the given market.
func (t *tradingDataService) MarketDepth(ctx context.Context, req *protoapi.MarketDepthRequest) (*protoapi.MarketDepthResponse, error) {

	// Query market depth statistics
	depth := t.marketDepthService.GetMarketDepth(req.MarketId, req.MaxDepth)

	lastOne := entities.OffsetPagination{Skip: 0, Limit: 1, Descending: true}
	ts, err := t.tradeService.GetByMarket(ctx, req.MarketId, lastOne)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}

	// Build market depth response, including last trade (if available)
	resp := &protoapi.MarketDepthResponse{
		Buy:            depth.Buy,
		MarketId:       depth.MarketId,
		Sell:           depth.Sell,
		SequenceNumber: depth.SequenceNumber,
	}
	if len(ts) > 0 {
		resp.LastTrade = ts[0].ToProto()
	}
	return resp, nil
}

// MarketDepthSubscribe opens a subscription to the MarketDepth service.
func (t *tradingDataService) MarketDepthSubscribe(
	req *protoapi.MarketDepthSubscribeRequest,
	srv protoapi.TradingDataService_MarketDepthSubscribeServer,
) error {

	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	_, err := validateMarketSQL(ctx, req.MarketId, t.marketService)
	if err != nil {
		return err // validateMarket already returns an API error, no additional wrapping needed
	}

	depthChan, ref := t.marketDepthService.ObserveDepth(
		ctx, t.Config.StreamRetries, req.MarketId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Depth subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observe(ctx, t.log, "MarketDepth", depthChan, ref, func(tr *vega.MarketDepth) error {
		return srv.Send(&protoapi.MarketDepthSubscribeResponse{
			MarketDepth: tr,
		})
	})
}

// MarketDepthUpdatesSubscribe opens a subscription to the MarketDepth Updates service.
func (t *tradingDataService) MarketDepthUpdatesSubscribe(
	req *protoapi.MarketDepthUpdatesSubscribeRequest,
	srv protoapi.TradingDataService_MarketDepthUpdatesSubscribeServer,
) error {

	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	_, err := validateMarketSQL(ctx, req.MarketId, t.marketService)
	if err != nil {
		return err // validateMarket already returns an API error, no additional wrapping needed
	}

	depthChan, ref := t.marketDepthService.ObserveDepthUpdates(
		ctx, t.Config.StreamRetries, req.MarketId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Depth updates subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observe(ctx, t.log, "MarketDepthUpdate", depthChan, ref, func(tr *vega.MarketDepthUpdate) error {
		return srv.Send(&protoapi.MarketDepthUpdatesSubscribeResponse{
			Update: tr,
		})
	})
}

/****************************** Positions **************************************/
func (t *tradingDataService) PositionsByParty(ctx context.Context, request *protoapi.PositionsByPartyRequest) (*protoapi.PositionsByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("PositionsByParty SQL")()

	var positions []entities.Position
	var err error

	if request.MarketId == "" && request.PartyId == "" {
		positions, err = t.positionService.GetAll(ctx)
	} else if request.MarketId == "" {
		positions, err = t.positionService.GetByParty(ctx, entities.NewPartyID(request.PartyId))
	} else if request.PartyId == "" {
		positions, err = t.positionService.GetByMarket(ctx, entities.NewMarketID(request.MarketId))
	} else {
		positions = make([]entities.Position, 1)
		positions[0], err = t.positionService.GetByMarketAndParty(ctx,
			entities.NewMarketID(request.MarketId),
			entities.NewPartyID(request.PartyId))

		// Don't error if there's no position for this party/market
		if errors.Is(err, sqlstore.ErrPositionNotFound) {
			err = nil
			positions = []entities.Position{}
		}
	}

	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetPositionsByParty, err)
	}

	out := make([]*vega.Position, len(positions))
	for i, position := range positions {
		out[i] = position.ToProto()
	}

	response := &protoapi.PositionsByPartyResponse{Positions: out}
	return response, nil
}

func (t *tradingDataService) PositionsSubscribe(
	req *protoapi.PositionsSubscribeRequest,
	srv protoapi.TradingDataService_PositionsSubscribeServer,
) error {

	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	positionsChan, ref := t.positionService.Observe(ctx, t.Config.StreamRetries, req.PartyId, req.MarketId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Positions subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observe(ctx, t.log, "Position", positionsChan, ref, func(position entities.Position) error {
		return srv.Send(&protoapi.PositionsSubscribeResponse{
			Position: position.ToProto(),
		})
	})

}

/****************************** Parties **************************************/
func (t *tradingDataService) Parties(ctx context.Context, _ *protoapi.PartiesRequest) (*protoapi.PartiesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Parties SQL")()
	parties, err := t.partyService.GetAll(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, ErrPartyServiceGetAll, err)
	}

	out := make([]*vega.Party, len(parties))
	for i, p := range parties {
		out[i] = p.ToProto()
	}

	return &protoapi.PartiesResponse{
		Parties: out,
	}, nil
}

func (t *tradingDataService) PartyByID(ctx context.Context, req *protoapi.PartyByIDRequest) (*protoapi.PartyByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("PartyByID SQL")()
	out := protoapi.PartyByIDResponse{}

	party, err := t.partyService.GetByID(ctx, req.PartyId)

	if errors.Is(err, sqlstore.ErrPartyNotFound) {
		return &out, nil
	}

	if errors.Is(err, sqlstore.ErrInvalidPartyID) {
		return &out, apiError(codes.InvalidArgument, ErrPartyServiceGetByID, err)
	}

	if err != nil {
		return nil, apiError(codes.Internal, ErrPartyServiceGetByID, err)
	}

	return &protoapi.PartyByIDResponse{
		Party: party.ToProto(),
	}, nil
}

/****************************** General **************************************/

func (t *tradingDataService) GetVegaTime(ctx context.Context, _ *protoapi.GetVegaTimeRequest) (*protoapi.GetVegaTimeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetVegaTime SQL")()
	b, err := t.blockService.GetLastBlock(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTimeServiceGetTimeNow, err)
	}

	return &protoapi.GetVegaTimeResponse{
		Timestamp: b.VegaTime.UnixNano(),
	}, nil
}

/****************************** Checkpoints **************************************/

func (t *tradingDataService) Checkpoints(ctx context.Context, _ *protoapi.CheckpointsRequest) (*protoapi.CheckpointsResponse, error) {
	checkpoints, err := t.checkpointService.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]*protoapi.Checkpoint, len(checkpoints))
	for i, cp := range checkpoints {
		out[i] = cp.ToProto()
	}

	return &protoapi.CheckpointsResponse{
		Checkpoints: out,
	}, nil
}

/****************************** Transfers **************************************/

func (t *tradingDataService) Transfers(ctx context.Context, req *protoapi.TransfersRequest) (*protoapi.TransfersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Transfers-SQL")()

	if len(req.Pubkey) <= 0 && (req.IsFrom || req.IsTo) {
		return nil, apiError(codes.InvalidArgument, errors.New("missing pubkey"))
	}

	if req.IsFrom && req.IsTo {
		return nil, apiError(codes.InvalidArgument, errors.New("request is for transfers to and from the same party"))
	}

	var transfers []entities.Transfer
	var err error
	if !req.IsFrom && !req.IsTo {
		transfers, _, err = t.transferService.GetAll(ctx, entities.CursorPagination{})
		if err != nil {
			return nil, apiError(codes.Internal, err)
		}
	} else if req.IsFrom || req.IsTo {

		if req.IsFrom {
			transfers, _, err = t.transferService.GetTransfersFromParty(ctx, entities.PartyID{ID: entities.ID(req.Pubkey)}, entities.CursorPagination{})
			if err != nil {
				return nil, apiError(codes.Internal, err)
			}
		}

		if req.IsTo {
			transfers, _, err = t.transferService.GetTransfersToParty(ctx, entities.PartyID{ID: entities.ID(req.Pubkey)}, entities.CursorPagination{})
			if err != nil {
				return nil, apiError(codes.Internal, err)
			}
		}
	}

	protoTransfers := make([]*eventspb.Transfer, 0, len(transfers))
	for _, transfer := range transfers {
		proto, err := transfer.ToProto(t.accountService)
		if err != nil {
			return nil, apiError(codes.Internal, err)
		}
		protoTransfers = append(protoTransfers, proto)
	}

	return &protoapi.TransfersResponse{
		Transfers: protoTransfers,
	}, nil
}

/****************************** Network Parameters **************************************/

func (t *tradingDataService) NetworkParameters(ctx context.Context, req *protoapi.NetworkParametersRequest) (*protoapi.NetworkParametersResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("NetworkParameters SQL")()
	nps, err := t.networkParameterService.GetAll(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	out := make([]*vega.NetworkParameter, len(nps))
	for i, np := range nps {
		out[i] = np.ToProto()
	}

	return &protoapi.NetworkParametersResponse{
		NetworkParameters: out,
	}, nil
}

/****************************** Candles **************************************/

func (t *tradingDataService) Candles(ctx context.Context,
	request *protoapi.CandlesRequest,
) (*protoapi.CandlesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Candles-SQL")()

	if request.Interval == vega.Interval_INTERVAL_UNSPECIFIED {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	from := vegatime.UnixNano(request.SinceTimestamp)
	interval, err := toV2IntervalString(request.Interval)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCandleServiceGetCandleData,
			fmt.Errorf("failed to get candles:%w", err))
	}

	exists, candleId, err := t.candleService.GetCandleIdForIntervalAndMarket(ctx, interval, request.MarketId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrCandleServiceGetCandleData,
			fmt.Errorf("failed to get candles:%w", err))
	}

	if !exists {
		return nil, apiError(codes.InvalidArgument, ErrCandleServiceGetCandleData,
			fmt.Errorf("candle does not exist for interval %s and market %s", interval, request.MarketId))
	}

	candles, _, err := t.candleService.GetCandleDataForTimeSpan(ctx, candleId, &from, nil, entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, ErrCandleServiceGetCandleData,
			fmt.Errorf("failed to get candles for interval:%w", err))
	}

	var protoCandles []*vega.Candle
	for _, candle := range candles {
		proto, err := candle.ToV1CandleProto(request.Interval)
		if err != nil {
			return nil, apiError(codes.Internal, ErrCandleServiceGetCandleData,
				fmt.Errorf("failed to convert candle to protobuf:%w", err))
		}

		protoCandles = append(protoCandles, proto)
	}

	return &protoapi.CandlesResponse{
		Candles: protoCandles,
	}, nil
}

func toV2IntervalString(interval vega.Interval) (string, error) {
	switch interval {
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

func (t *tradingDataService) CandlesSubscribe(req *protoapi.CandlesSubscribeRequest,
	srv protoapi.TradingDataService_CandlesSubscribeServer,
) error {
	defer metrics.StartActiveSubscriptionCountGRPC("Candles")()
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	interval, err := toV2IntervalString(req.Interval)
	if err != nil {
		return apiError(codes.InvalidArgument, ErrStreamInternal,
			fmt.Errorf("subscribing to candles:%w", err))
	}

	exists, candleId, err := t.candleService.GetCandleIdForIntervalAndMarket(ctx, interval, req.MarketId)
	if err != nil {
		return apiError(codes.InvalidArgument, ErrStreamInternal,
			fmt.Errorf("subscribing to candles:%w", err))
	}

	if !exists {
		return apiError(codes.InvalidArgument, ErrStreamInternal,
			fmt.Errorf("candle does not exist for interval %s and market %s", interval, req.MarketId))
	}

	ref, candlesChan, err := t.candleService.Subscribe(ctx, candleId)
	if err != nil {
		return apiError(codes.Internal, ErrStreamInternal,
			fmt.Errorf("subscribing to candles:%w", err))
	}

	for {
		select {
		case candle, ok := <-candlesChan:

			if !ok {
				err = ErrChannelClosed
				return apiError(codes.Internal, err)
			}
			proto, err := candle.ToV1CandleProto(req.Interval)
			if err != nil {
				return apiError(codes.Internal, ErrStreamInternal, err)
			}

			resp := &protoapi.CandlesSubscribeResponse{
				Candle: proto,
			}
			if err = srv.Send(resp); err != nil {
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			err := t.candleService.Unsubscribe(ref)
			if err != nil {
				t.log.Errorf("failed to unsubscribe from candle updates:%s", err)
			}
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		}
	}
}

/****************************** Governance **************************************/

func (t *tradingDataService) GetProposals(ctx context.Context, req *protoapi.GetProposalsRequest,
) (*protoapi.GetProposalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetProposals SQL")()

	inState := proposalState(req.SelectInState)

	proposals, _, err := t.governanceService.GetProposals(ctx, inState, nil, nil,
		entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	governanceData, err := t.proposalListToGovernanceData(ctx, proposals)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.GetProposalsResponse{Data: governanceData}, nil
}

func (t *tradingDataService) GetProposalsByParty(ctx context.Context,
	req *protoapi.GetProposalsByPartyRequest,
) (*protoapi.GetProposalsByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetProposalsByParty SQL")()

	inState := proposalState(req.SelectInState)

	proposals, _, err := t.governanceService.GetProposals(ctx, inState, &req.PartyId, nil,
		entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	governanceData, err := t.proposalListToGovernanceData(ctx, proposals)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.GetProposalsByPartyResponse{
		Data: governanceData,
	}, nil
}

func (t *tradingDataService) GetProposalByID(ctx context.Context,
	req *protoapi.GetProposalByIDRequest,
) (*protoapi.GetProposalByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetProposalByID SQL")()

	proposal, err := t.governanceService.GetProposalByID(ctx, req.ProposalId)
	if errors.Is(err, sqlstore.ErrProposalNotFound) {
		return nil, apiError(codes.NotFound, ErrMissingProposalID, err)
	} else if err != nil {
		return nil, apiError(codes.Internal, ErrNotMapped, err)
	}

	gd, err := t.proposalToGovernanceData(ctx, proposal)
	if err != nil {
		return nil, apiError(codes.Internal, ErrNotMapped, err)
	}

	return &protoapi.GetProposalByIDResponse{Data: gd}, nil
}

func (t *tradingDataService) GetProposalByReference(ctx context.Context,
	req *protoapi.GetProposalByReferenceRequest,
) (*protoapi.GetProposalByReferenceResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetProposalByID SQL")()

	proposal, err := t.governanceService.GetProposalByReference(ctx, req.Reference)
	if errors.Is(err, sqlstore.ErrProposalNotFound) {
		return nil, apiError(codes.NotFound, ErrMissingProposalReference, err)
	} else if err != nil {
		return nil, apiError(codes.Internal, ErrNotMapped, err)
	}

	gd, err := t.proposalToGovernanceData(ctx, proposal)
	if err != nil {
		return nil, apiError(codes.Internal, ErrNotMapped, err)
	}

	return &protoapi.GetProposalByReferenceResponse{Data: gd}, nil
}

func (t *tradingDataService) GetVotesByParty(ctx context.Context,
	req *protoapi.GetVotesByPartyRequest,
) (*protoapi.GetVotesByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetVotesByParty SQL")()

	votes, err := t.governanceService.GetVotesByParty(ctx, req.PartyId)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.GetVotesByPartyResponse{Votes: voteListToProto(votes)}, nil
}

func (t *tradingDataService) GetNewMarketProposals(ctx context.Context,
	req *protoapi.GetNewMarketProposalsRequest,
) (*protoapi.GetNewMarketProposalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNewMarketProposals SQL")()

	inState := proposalState(req.SelectInState)
	proposals, _, err := t.governanceService.GetProposals(ctx, inState, nil, &entities.ProposalTypeNewMarket,
		entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	governanceData, err := t.proposalListToGovernanceData(ctx, proposals)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	return &protoapi.GetNewMarketProposalsResponse{Data: governanceData}, nil
}

func (t *tradingDataService) GetUpdateMarketProposals(ctx context.Context,
	req *protoapi.GetUpdateMarketProposalsRequest,
) (*protoapi.GetUpdateMarketProposalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetUpdateMarketProposals SQL")()

	inState := proposalState(req.SelectInState)
	proposals, _, err := t.governanceService.GetProposals(ctx, inState, nil, &entities.ProposalTypeUpdateMarket,
		entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	governanceData, err := t.proposalListToGovernanceData(ctx, proposals)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	return &protoapi.GetUpdateMarketProposalsResponse{Data: governanceData}, nil
}

func (t *tradingDataService) GetNetworkParametersProposals(ctx context.Context,
	req *protoapi.GetNetworkParametersProposalsRequest,
) (*protoapi.GetNetworkParametersProposalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNetworkParametersProposals SQL")()

	inState := proposalState(req.SelectInState)

	proposals, _, err := t.governanceService.GetProposals(ctx, inState, nil, &entities.ProposalTypeUpdateNetworkParameter,
		entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	governanceData, err := t.proposalListToGovernanceData(ctx, proposals)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.GetNetworkParametersProposalsResponse{Data: governanceData}, nil
}

func (t *tradingDataService) GetNewAssetProposals(ctx context.Context,
	req *protoapi.GetNewAssetProposalsRequest,
) (*protoapi.GetNewAssetProposalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNewAssetProposals SQL")()

	inState := proposalState(req.SelectInState)

	proposals, _, err := t.governanceService.GetProposals(ctx, inState, nil, &entities.ProposalTypeNewAsset,
		entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	governanceData, err := t.proposalListToGovernanceData(ctx, proposals)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.GetNewAssetProposalsResponse{Data: governanceData}, nil
}

func (t *tradingDataService) GetNewFreeformProposals(ctx context.Context,
	req *protoapi.GetNewFreeformProposalsRequest,
) (*protoapi.GetNewFreeformProposalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNewFreeformProposals SQL")()

	inState := proposalState(req.SelectInState)

	proposals, _, err := t.governanceService.GetProposals(ctx, inState, nil, &entities.ProposalTypeNewFreeform,
		entities.CursorPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	governanceData, err := t.proposalListToGovernanceData(ctx, proposals)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.GetNewFreeformProposalsResponse{Data: governanceData}, nil
}

func proposalState(protoState *protoapi.OptionalProposalState) *entities.ProposalState {
	var s *entities.ProposalState
	if protoState != nil {
		state := entities.ProposalState(protoState.Value)
		s = &state
	}
	return s
}

func (t *tradingDataService) proposalListToGovernanceData(ctx context.Context, proposals []entities.Proposal) ([]*vega.GovernanceData, error) {
	governanceData := make([]*vega.GovernanceData, len(proposals))
	for i, proposal := range proposals {
		gd, err := t.proposalToGovernanceData(ctx, proposal)
		if err != nil {
			return nil, err
		}
		governanceData[i] = gd
	}
	return governanceData, nil
}

func (t *tradingDataService) proposalToGovernanceData(ctx context.Context, proposal entities.Proposal) (*vega.GovernanceData, error) {
	yesVotes, err := t.governanceService.GetYesVotesForProposal(ctx, proposal.ID.String())
	if err != nil {
		return nil, err
	}
	protoYesVotes := voteListToProto(yesVotes)

	noVotes, err := t.governanceService.GetNoVotesForProposal(ctx, proposal.ID.String())
	if err != nil {
		return nil, err
	}
	protoNoVotes := voteListToProto(noVotes)

	gd := vega.GovernanceData{
		Proposal: proposal.ToProto(),
		Yes:      protoYesVotes,
		No:       protoNoVotes,
	}
	return &gd, nil
}

func voteListToProto(votes []entities.Vote) []*vega.Vote {
	protoVotes := make([]*vega.Vote, len(votes))
	for j, vote := range votes {
		protoVotes[j] = vote.ToProto()
	}
	return protoVotes
}

func (t *tradingDataService) ObserveGovernance(
	_ *protoapi.ObserveGovernanceRequest,
	stream protoapi.TradingDataService_ObserveGovernanceServer,
) error {

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming governance updates")
	}
	ch, ref := t.governanceService.ObserveProposals(ctx, t.Config.StreamRetries, nil)

	return observe(ctx, t.log, "Governance", ch, ref, func(proposal entities.Proposal) error {
		gd, err := t.proposalToGovernanceData(ctx, proposal)
		if err != nil {
			return err
		}
		return stream.Send(&protoapi.ObserveGovernanceResponse{
			Data: gd,
		})
	})
}

func (t *tradingDataService) ObservePartyProposals(
	in *protoapi.ObservePartyProposalsRequest,
	stream protoapi.TradingDataService_ObservePartyProposalsServer,
) error {

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming party proposals")
	}

	ch, ref := t.governanceService.ObserveProposals(ctx, t.Config.StreamRetries, &in.PartyId)
	return observe(ctx, t.log, "PartyProposal", ch, ref, func(proposal entities.Proposal) error {
		gd, err := t.proposalToGovernanceData(ctx, proposal)
		if err != nil {
			return err
		}
		return stream.Send(&protoapi.ObservePartyProposalsResponse{
			Data: gd,
		})
	})
}

func (t *tradingDataService) ObservePartyVotes(
	in *protoapi.ObservePartyVotesRequest,
	stream protoapi.TradingDataService_ObservePartyVotesServer,
) error {

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming party votes")
	}
	ch, ref := t.governanceService.ObservePartyVotes(ctx, t.Config.StreamRetries, in.PartyId)

	return observe(ctx, t.log, "PartyVote", ch, ref, func(vote entities.Vote) error {
		return stream.Send(&protoapi.ObservePartyVotesResponse{
			Vote: vote.ToProto(),
		})
	})
}

func (t *tradingDataService) ObserveProposalVotes(
	in *protoapi.ObserveProposalVotesRequest,
	stream protoapi.TradingDataService_ObserveProposalVotesServer,
) error {

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming proposal votes")
	}
	ch, ref := t.governanceService.ObserveProposalVotes(ctx, t.Config.StreamRetries, in.ProposalId)

	return observe(ctx, t.log, "ProposalVote", ch, ref, func(p entities.Vote) error {
		return stream.Send(&protoapi.ObserveProposalVotesResponse{
			Vote: p.ToProto(),
		})
	})
}

/****************************** Epochs **************************************/

func (t *tradingDataService) GetEpoch(ctx context.Context, req *protoapi.GetEpochRequest) (*protoapi.GetEpochResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetEpoch SQL")()

	var epoch entities.Epoch
	var err error

	if req.GetId() == 0 {
		epoch, err = t.epochService.GetCurrent(ctx)
	} else {
		epoch, err = t.epochService.Get(ctx, int64(req.GetId()))
	}

	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	protoEpoch := epoch.ToProto()

	delegations, err := t.delegationService.Get(ctx, nil, nil, &epoch.ID, nil)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	protoDelegations := make([]*vega.Delegation, len(delegations))
	for i, delegation := range delegations {
		protoDelegations[i] = delegation.ToProto()
	}
	protoEpoch.Delegations = protoDelegations

	nodes, err := t.nodeService.GetNodes(ctx, uint64(epoch.ID))
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	protoNodes := make([]*vega.Node, len(nodes))
	for i, node := range nodes {
		protoNodes[i] = node.ToProto()
	}

	protoEpoch.Validators = protoNodes

	return &protoapi.GetEpochResponse{
		Epoch: protoEpoch,
	}, nil
}

/****************************** Delegations **************************************/

func (t *tradingDataService) Delegations(ctx context.Context,
	req *protoapi.DelegationsRequest,
) (*protoapi.DelegationsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Delegations SQL")()

	var delegations []entities.Delegation
	var err error

	p := defaultPaginationV2
	if req.Pagination != nil {
		p = toEntityPagination(req.Pagination)
	}

	var epochID *int64
	var partyID *string
	var nodeID *string

	if req.EpochSeq != "" {
		epochNum, err := strconv.ParseInt(req.EpochSeq, 10, 64)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, err)
		}
		epochID = &epochNum
	}

	if req.Party != "" {
		partyID = &req.Party
	}

	if req.NodeId != "" {
		nodeID = &req.NodeId
	}

	delegations, err = t.delegationService.Get(ctx, partyID, nodeID, epochID, &p)

	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	protoDelegations := make([]*vega.Delegation, len(delegations))
	for i, delegation := range delegations {
		protoDelegations[i] = delegation.ToProto()
	}

	return &protoapi.DelegationsResponse{
		Delegations: protoDelegations,
	}, nil
}

func (t *tradingDataService) ObserveDelegations(
	req *protoapi.ObserveDelegationsRequest,
	stream protoapi.TradingDataService_ObserveDelegationsServer,
) error {

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming delegation updates")
	}
	ch, ref := t.delegationService.Observe(ctx, t.Config.StreamRetries, req.Party, req.NodeId)

	return observe(ctx, t.log, "Delegations", ch, ref, func(delegation entities.Delegation) error {
		return stream.Send(&protoapi.ObserveDelegationsResponse{
			Delegation: delegation.ToProto(),
		})
	})
}

/****************************** Rewards **************************************/

func (t *tradingDataService) GetRewards(ctx context.Context,
	req *protoapi.GetRewardsRequest,
) (*protoapi.GetRewardsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetRewards-SQL")()
	if len(req.PartyId) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrGetRewards)
	}

	p := defaultPaginationV2
	if req.Pagination != nil {
		p = toEntityPagination(req.Pagination)
	}

	var rewards []entities.Reward
	var err error

	if len(req.AssetId) <= 0 {
		rewards, err = t.rewardService.GetByOffset(ctx, &req.PartyId, nil, &p)
	} else {
		rewards, err = t.rewardService.GetByOffset(ctx, &req.PartyId, &req.AssetId, &p)
	}

	if err != nil {
		return nil, apiError(codes.Internal, ErrGetRewards, err)
	}

	protoRewards := make([]*vega.Reward, len(rewards))
	for i, reward := range rewards {
		protoRewards[i] = reward.ToProto()
	}

	return &protoapi.GetRewardsResponse{Rewards: protoRewards}, nil
}

func (t *tradingDataService) GetRewardSummaries(ctx context.Context,
	req *protoapi.GetRewardSummariesRequest,
) (*protoapi.GetRewardSummariesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetRewardSummaries-SQL")()

	if len(req.PartyId) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrTradeServiceGetByParty)
	}

	var summaries []entities.RewardSummary
	var err error

	if len(req.AssetId) <= 0 {
		summaries, err = t.rewardService.GetSummaries(ctx, &req.PartyId, nil)
	} else {
		summaries, err = t.rewardService.GetSummaries(ctx, &req.PartyId, &req.AssetId)
	}

	if err != nil {
		return nil, apiError(codes.Internal, ErrGetRewards, err)
	}

	protoSummaries := make([]*vega.RewardSummary, len(summaries))
	for i, summary := range summaries {
		protoSummaries[i] = summary.ToProto()
	}

	return &protoapi.GetRewardSummariesResponse{Summaries: protoSummaries}, nil
}

func (t *tradingDataService) ObserveRewards(req *protoapi.ObserveRewardsRequest,
	stream protoapi.TradingDataService_ObserveRewardsServer,
) error {

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()
	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("starting streaming reward updates")
	}
	ch, ref := t.rewardService.Observe(ctx, t.Config.StreamRetries, req.AssetId, req.Party)

	return observe(ctx, t.log, "Reward", ch, ref, func(reward entities.Reward) error {
		return stream.Send(&protoapi.ObserveRewardsResponse{
			Reward: reward.ToProto(),
		})
	})
}

/****************************** Trades **************************************/
// TradesByParty provides a list of trades for the given party.
// OffsetPagination: Optional. If not provided, defaults are used.
func (t *tradingDataService) TradesByParty(ctx context.Context,
	req *protoapi.TradesByPartyRequest,
) (*protoapi.TradesByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("TradesByParty-SQL")()

	p := defaultEntityPagination
	if req.Pagination != nil {
		p = toEntityPagination(req.Pagination)
	}

	trades, err := t.tradeService.GetByParty(ctx, req.PartyId, &req.MarketId, p)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByParty, err)
	}

	protoTrades := tradesToProto(trades)

	return &protoapi.TradesByPartyResponse{Trades: protoTrades}, nil
}

func tradesToProto(trades []entities.Trade) []*vega.Trade {
	protoTrades := []*vega.Trade{}
	for _, trade := range trades {
		protoTrades = append(protoTrades, trade.ToProto())
	}
	return protoTrades
}

// TradesByOrder provides a list of the trades that correspond to a given order.
func (t *tradingDataService) TradesByOrder(ctx context.Context,
	req *protoapi.TradesByOrderRequest,
) (*protoapi.TradesByOrderResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("TradesByOrder-SQL")()

	trades, err := t.tradeService.GetByOrderID(ctx, req.OrderId, nil, defaultEntityPagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByOrderID, err)
	}

	protoTrades := tradesToProto(trades)

	return &protoapi.TradesByOrderResponse{Trades: protoTrades}, nil
}

// TradesByMarket provides a list of trades for a given market.
// OffsetPagination: Optional. If not provided, defaults are used.
func (t *tradingDataService) TradesByMarket(ctx context.Context, req *protoapi.TradesByMarketRequest) (*protoapi.TradesByMarketResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("TradesByMarket-SQL")()

	p := defaultEntityPagination
	if req.Pagination != nil {
		p = toEntityPagination(req.Pagination)
	}

	trades, err := t.tradeService.GetByMarket(ctx, req.MarketId, p)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}

	protoTrades := tradesToProto(trades)
	return &protoapi.TradesByMarketResponse{
		Trades: protoTrades,
	}, nil
}

// LastTrade provides the last trade for the given market.
func (t *tradingDataService) LastTrade(ctx context.Context,
	req *protoapi.LastTradeRequest,
) (*protoapi.LastTradeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("LastTrade-SQL")()

	if len(req.MarketId) <= 0 {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingMarketID)
	}

	p := entities.OffsetPagination{
		Skip:       0,
		Limit:      1,
		Descending: true,
	}

	trades, err := t.tradeService.GetByMarket(ctx, req.MarketId, p)
	if err != nil {
		return nil, apiError(codes.Internal, ErrTradeServiceGetByMarket, err)
	}

	protoTrades := tradesToProto(trades)

	if len(protoTrades) > 0 && protoTrades[0] != nil {
		return &protoapi.LastTradeResponse{Trade: protoTrades[0]}, nil
	}
	// No trades found on the market yet (and no errors)
	// this can happen at the beginning of a new market
	return &protoapi.LastTradeResponse{}, nil
}

// TradesSubscribe opens a subscription to the Trades service.
func (t *tradingDataService) TradesSubscribe(req *protoapi.TradesSubscribeRequest,
	srv protoapi.TradingDataService_TradesSubscribeServer,
) error {

	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	var (
		marketID, partyID *string
	)
	if len(req.MarketId) > 0 {
		marketID = &req.MarketId
	}
	if len(req.PartyId) > 0 {
		partyID = &req.PartyId
	}

	tradesChan, ref := t.tradeService.Observe(ctx, t.Config.StreamRetries, marketID, partyID)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Trades subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "Trade", tradesChan, ref, func(trades []*entities.Trade) error {
		out := make([]*pbtypes.Trade, 0, len(trades))
		for _, v := range trades {
			out = append(out, v.ToProto())
		}
		return srv.Send(&protoapi.TradesSubscribeResponse{Trades: out})
	})
}

/****************************** Orders **************************************/

// OrdersSubscribe opens a subscription to the Orders service.
// MarketID: Optional.
// PartyID: Optional.
func (t *tradingDataService) OrdersSubscribe(
	req *protoapi.OrdersSubscribeRequest, srv protoapi.TradingDataService_OrdersSubscribeServer,
) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	var (
		marketID, partyID *string
	)

	if len(req.MarketId) > 0 {
		marketID = &req.MarketId
	}
	if len(req.PartyId) > 0 {
		partyID = &req.PartyId
	}

	ordersChan, ref := t.orderService.ObserveOrders(ctx, t.Config.StreamRetries, marketID, partyID)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Orders subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observeBatch(ctx, t.log, "Order", ordersChan, ref, func(orders []entities.Order) error {
		out := make([]*pbtypes.Order, 0, len(orders))
		for _, v := range orders {
			out = append(out, v.ToProto())
		}
		return srv.Send(&protoapi.OrdersSubscribeResponse{Orders: out})
	})
}

func (t *tradingDataService) OrderByID(ctx context.Context, req *protoapi.OrderByIDRequest) (*protoapi.OrderByIDResponse, error) {
	if len(req.OrderId) == 0 {
		return nil, ErrMissingOrderIDParameter
	}

	version := int32(req.Version)
	order, err := t.orderService.GetOrder(ctx, req.OrderId, &version)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	resp := &protoapi.OrderByIDResponse{Order: order.ToProto()}
	return resp, nil
}

// OrderVersionsByID returns all versions of the order by its orderID
func (t *tradingDataService) OrderVersionsByID(
	ctx context.Context,
	in *protoapi.OrderVersionsByIDRequest,
) (*protoapi.OrderVersionsByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrderVersionsByID")()

	p := defaultPaginationV2
	if in.Pagination != nil {
		p = toEntityPagination(in.Pagination)
	}

	orders, err := t.orderService.GetAllVersionsByOrderID(ctx, in.OrderId, p)
	pbOrders := make([]*vega.Order, len(orders))
	for i, order := range orders {
		pbOrders[i] = order.ToProto()
	}

	if err == nil {
		return &protoapi.OrderVersionsByIDResponse{
			Orders: pbOrders,
		}, nil
	}
	return nil, err
}

func (t *tradingDataService) OrderByMarketAndID(ctx context.Context,
	req *protoapi.OrderByMarketAndIDRequest,
) (*protoapi.OrderByMarketAndIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrderByMarketAndID-SQL")()

	// This function is no longer needed; IDs are globally unique now, but keep it for compatibility for now
	if len(req.OrderId) == 0 {
		return nil, ErrMissingOrderIDParameter
	}

	order, err := t.orderService.GetOrder(ctx, req.OrderId, nil)
	if err != nil {
		return nil, ErrOrderNotFound
	}

	resp := &protoapi.OrderByMarketAndIDResponse{Order: order.ToProto()}
	return resp, nil
}

// OrderByReference provides the (possibly not yet accepted/rejected) order.
func (t *tradingDataService) OrderByReference(ctx context.Context, req *protoapi.OrderByReferenceRequest) (*protoapi.OrderByReferenceResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrderByReference-SQL")()

	orders, err := t.orderService.GetByReference(ctx, req.Reference, entities.OffsetPagination{})
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByReference, err)
	}

	if len(orders) == 0 {
		return nil, ErrOrderNotFound
	}
	return &protoapi.OrderByReferenceResponse{
		Order: orders[0].ToProto(),
	}, nil
}

func (t *tradingDataService) OrdersByParty(ctx context.Context,
	req *protoapi.OrdersByPartyRequest,
) (*protoapi.OrdersByPartyResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrdersByParty-SQL")()

	p := defaultPaginationV2
	if req.Pagination != nil {
		p = toEntityPagination(req.Pagination)
	}

	orders, err := t.orderService.GetByParty(ctx, req.PartyId, p)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, ErrOrderServiceGetByParty, err)
	}

	pbOrders := make([]*vega.Order, len(orders))
	for i, order := range orders {
		pbOrders[i] = order.ToProto()
	}

	return &protoapi.OrdersByPartyResponse{
		Orders: pbOrders,
	}, nil
}

// OrdersByMarket provides a list of orders for a given market.
// CursorPagination: Optional. If not provided, defaults are used.
// Returns a list of orders sorted by timestamp descending (most recent first).
func (t *tradingDataService) OrdersByMarket(ctx context.Context,
	request *protoapi.OrdersByMarketRequest,
) (*protoapi.OrdersByMarketResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OrdersByMarket-SQL")()

	p := defaultPaginationV2
	if request.Pagination != nil {
		p = toEntityPagination(request.Pagination)
	}

	orders, err := t.orderService.GetByMarket(ctx, request.MarketId, p)
	if err != nil {
		return nil, apiError(codes.Internal, ErrOrderServiceGetByMarket, err)
	}

	pbOrders := make([]*vega.Order, len(orders))
	for i, order := range orders {
		pbOrders[i] = order.ToProto()
	}

	response := &protoapi.OrdersByMarketResponse{}
	if len(orders) > 0 {
		response.Orders = pbOrders
	}

	return response, nil
}

func toEntityPagination(pagination *protoapi.Pagination) entities.OffsetPagination {
	return entities.OffsetPagination{
		Skip:       pagination.Skip,
		Limit:      pagination.Limit,
		Descending: pagination.Descending,
	}
}

func (t *tradingDataService) AssetByID(ctx context.Context, req *protoapi.AssetByIDRequest) (*protoapi.AssetByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("AssetByID-SQL")()
	if len(req.Id) <= 0 {
		return nil, apiError(codes.InvalidArgument, errors.New("missing ID"))
	}

	asset, err := t.assetService.GetByID(ctx, req.Id)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	return &protoapi.AssetByIDResponse{
		Asset: asset.ToProto(),
	}, nil
}

func (t *tradingDataService) Assets(ctx context.Context, _ *protoapi.AssetsRequest) (*protoapi.AssetsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Assets-SQL")()

	assets, _ := t.assetService.GetAll(ctx)

	out := make([]*vega.Asset, 0, len(assets))
	for _, v := range assets {
		out = append(out, v.ToProto())
	}
	return &protoapi.AssetsResponse{
		Assets: out,
	}, nil
}

/****************************** Accounts **************************************/

func isValidAccountType(accountType vega.AccountType, validAccountTypes ...vega.AccountType) bool {
	for _, vt := range validAccountTypes {
		if accountType == vt {
			return true
		}
	}

	return false
}

func (t *tradingDataService) PartyAccounts(ctx context.Context, req *protoapi.PartyAccountsRequest) (*protoapi.PartyAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("PartyAccounts_SQL")()

	// This is just nicer to read and update if the list of valid account types change than multiple AND statements
	if !isValidAccountType(req.Type, vega.AccountType_ACCOUNT_TYPE_GENERAL, vega.AccountType_ACCOUNT_TYPE_MARGIN,
		vega.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW, vega.AccountType_ACCOUNT_TYPE_BOND, vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED) {
		return nil, errors.New("invalid type for query, only GENERAL, MARGIN, LOCK_WITHDRAW AND BOND accounts for a party supported")
	}

	pagination := entities.OffsetPagination{}

	filter := entities.AccountFilter{
		AssetID:      toAccountsFilterAsset(req.Asset),
		PartyIDs:     toAccountsFilterParties(req.PartyId),
		AccountTypes: toAccountsFilterAccountTypes(req.Type),
		MarketIDs:    toAccountsFilterMarkets(req.MarketId),
	}

	accountBalances, err := t.accountService.QueryBalancesV1(ctx, filter, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetPartyAccounts, err)
	}

	return &protoapi.PartyAccountsResponse{
		Accounts: accountBalancesToProtoAccountList(accountBalances),
	}, nil
}

func toAccountsFilterAccountTypes(accountTypes ...vega.AccountType) []vega.AccountType {
	accountTypesProto := make([]vega.AccountType, 0)

	for _, accountType := range accountTypes {
		if accountType == vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
			return nil
		}

		accountTypesProto = append(accountTypesProto, accountType)
	}

	return accountTypesProto
}

func accountBalancesToProtoAccountList(accounts []entities.AccountBalance) []*vega.Account {
	accountsProto := make([]*vega.Account, 0, len(accounts))

	for _, acc := range accounts {
		accountsProto = append(accountsProto, acc.ToProto())
	}

	return accountsProto
}

func toAccountsFilterAsset(assetID string) entities.AssetID {
	return entities.NewAssetID(assetID)
}

func toAccountsFilterParties(partyIDs ...string) []entities.PartyID {
	parties := make([]entities.PartyID, 0, len(partyIDs))
	for _, idStr := range partyIDs {
		if idStr == "" {
			continue
		}
		parties = append(parties, entities.NewPartyID(idStr))
	}

	return parties
}

func toAccountsFilterMarkets(marketIDs ...string) []entities.MarketID {
	markets := make([]entities.MarketID, 0, len(marketIDs))
	for _, idStr := range marketIDs {
		if idStr == "" {
			continue
		}
		market := entities.NewMarketID(idStr)
		markets = append(markets, market)
	}

	return markets
}

func (t *tradingDataService) MarketAccounts(ctx context.Context,
	req *protoapi.MarketAccountsRequest,
) (*protoapi.MarketAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketAccounts")()

	filter := entities.AccountFilter{
		AssetID:   toAccountsFilterAsset(req.Asset),
		MarketIDs: toAccountsFilterMarkets(req.MarketId),
		AccountTypes: toAccountsFilterAccountTypes(
			vega.AccountType_ACCOUNT_TYPE_INSURANCE,
			vega.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY,
		),
	}

	pagination := entities.OffsetPagination{}

	accountBalances, err := t.accountService.QueryBalancesV1(ctx, filter, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetMarketAccounts, err)
	}

	return &protoapi.MarketAccountsResponse{
		Accounts: accountBalancesToProtoAccountList(accountBalances),
	}, nil
}

func (t *tradingDataService) FeeInfrastructureAccounts(ctx context.Context,
	req *protoapi.FeeInfrastructureAccountsRequest,
) (*protoapi.FeeInfrastructureAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("FeeInfrastructureAccounts")()

	filter := entities.AccountFilter{
		AssetID: toAccountsFilterAsset(req.Asset),
		AccountTypes: toAccountsFilterAccountTypes(
			vega.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE,
		),
	}
	pagination := entities.OffsetPagination{}

	accountBalances, err := t.accountService.QueryBalancesV1(ctx, filter, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetFeeInfrastructureAccounts, err)
	}
	return &protoapi.FeeInfrastructureAccountsResponse{
		Accounts: accountBalancesToProtoAccountList(accountBalances),
	}, nil
}

func (t *tradingDataService) GlobalRewardPoolAccounts(ctx context.Context,
	req *protoapi.GlobalRewardPoolAccountsRequest,
) (*protoapi.GlobalRewardPoolAccountsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GloabRewardPoolAccounts")()
	filter := entities.AccountFilter{
		AssetID: toAccountsFilterAsset(req.Asset),
		AccountTypes: toAccountsFilterAccountTypes(
			vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
		),
	}
	pagination := entities.OffsetPagination{}

	accountBalances, err := t.accountService.QueryBalancesV1(ctx, filter, pagination)
	if err != nil {
		return nil, apiError(codes.Internal, ErrAccountServiceGetGlobalRewardPoolAccounts, err)
	}
	return &protoapi.GlobalRewardPoolAccountsResponse{
		Accounts: accountBalancesToProtoAccountList(accountBalances),
	}, nil
}

// AccountsSubscribe opens a subscription to the Accounts service.
func (t *tradingDataService) AccountsSubscribe(req *protoapi.AccountsSubscribeRequest,
	srv protoapi.TradingDataService_AccountsSubscribeServer,
) error {

	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	accountsChan, ref := t.accountService.ObserveAccountBalances(
		ctx, t.Config.StreamRetries, req.MarketId, req.PartyId, req.Asset, req.Type)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Accounts subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observe(ctx, t.log, "Accounts", accountsChan, ref, func(account entities.AccountBalance) error {
		return srv.Send(&protoapi.AccountsSubscribeResponse{
			Account: account.ToProto(),
		})
	})
}

/****************************** Market Data **************************************/

// MarketDataByID provides market data for the given ID.
func (t *tradingDataService) MarketDataByID(ctx context.Context, req *protoapi.MarketDataByIDRequest) (*protoapi.MarketDataByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketDataByID_SQL")()

	// validate the market exist
	if req.MarketId != "" {
		_, err := t.marketService.GetByID(ctx, req.MarketId)
		if err != nil {
			return nil, apiError(codes.InvalidArgument, ErrInvalidMarketID, err)
		}
	}

	md, err := t.marketDataService.GetMarketDataByID(ctx, req.MarketId)
	if err != nil {
		return nil, apiError(codes.Internal, ErrMarketServiceGetMarketData, err)
	}
	return &protoapi.MarketDataByIDResponse{
		MarketData: md.ToProto(),
	}, nil
}

// MarketsData provides all market data for all markets on this network.
func (t *tradingDataService) MarketsData(ctx context.Context, _ *protoapi.MarketsDataRequest) (*protoapi.MarketsDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketsData_SQL")()
	mds, _ := t.marketDataService.GetMarketsData(ctx)

	mdptrs := make([]*vega.MarketData, 0, len(mds))
	for _, v := range mds {
		mdptrs = append(mdptrs, v.ToProto())
	}

	return &protoapi.MarketsDataResponse{
		MarketsData: mdptrs,
	}, nil
}

// MarketsDataSubscribe opens a subscription to market data provided by the markets service.
func (t *tradingDataService) MarketsDataSubscribe(req *protoapi.MarketsDataSubscribeRequest,
	srv protoapi.TradingDataService_MarketsDataSubscribeServer,
) error {
	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	marketsDataChan, ref := t.marketDataService.ObserveMarketData(ctx, t.Config.StreamRetries, req.MarketId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Markets data subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observe(ctx, t.log, "MarketsData", marketsDataChan, ref, func(md *entities.MarketData) error {
		return srv.Send(&protoapi.MarketsDataSubscribeResponse{
			MarketData: md.ToProto(),
		})
	})
}

/****************************** Markets **************************************/

// MarketByID provides the given market.
func (t *tradingDataService) MarketByID(ctx context.Context, req *protoapi.MarketByIDRequest) (*protoapi.MarketByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarketByID_SQL")()

	mkt, err := validateMarketSQL(ctx, req.MarketId, t.marketService)
	if err != nil {
		return nil, err // validateMarket already returns an API error, no need to additionally wrap
	}

	return &protoapi.MarketByIDResponse{
		Market: mkt,
	}, nil
}

func validateMarketSQL(ctx context.Context, marketID string, marketsStore *service.Markets) (*vega.Market, error) {
	if len(marketID) == 0 {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingMarketID)
	}

	market, err := marketsStore.GetByID(ctx, marketID)
	if err != nil {
		// We return nil for error as we do not want
		// to return an error when a market is not found
		// but just a nil value.
		return nil, nil
	}

	mkt := market.ToProto()
	return mkt, nil
}

// Markets provides a list of all current markets that exist on the VEGA platform.
func (t *tradingDataService) Markets(ctx context.Context, _ *protoapi.MarketsRequest) (*protoapi.MarketsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Markets_SQL")()
	markets, err := t.marketService.GetAll(ctx, entities.OffsetPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, ErrMarketServiceGetMarkets, err)
	}

	results := make([]*vega.Market, 0, len(markets))
	for _, m := range markets {
		mkt := m.ToProto()
		results = append(results, mkt)
	}

	return &protoapi.MarketsResponse{
		Markets: results,
	}, nil
}

func (t *tradingDataService) Deposit(ctx context.Context, req *protoapi.DepositRequest) (*protoapi.DepositResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Deposit SQL")()
	if len(req.Id) <= 0 {
		return nil, ErrMissingDepositID
	}
	deposit, err := t.depositService.GetByID(ctx, req.Id)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}
	return &protoapi.DepositResponse{
		Deposit: deposit.ToProto(),
	}, nil
}

func (t *tradingDataService) Deposits(ctx context.Context, req *protoapi.DepositsRequest) (*protoapi.DepositsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Deposits SQL")()
	if len(req.PartyId) <= 0 {
		return nil, ErrMissingPartyID
	}

	// current API doesn't support pagination, but we will need to support it for v2
	deposits, _, err := t.depositService.GetByParty(ctx, req.PartyId, false, entities.OffsetPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	out := make([]*vega.Deposit, 0, len(deposits))
	for _, v := range deposits {
		out = append(out, v.ToProto())
	}
	return &protoapi.DepositsResponse{
		Deposits: out,
	}, nil
}

/****************************** Market Data **************************************/

func (t *tradingDataService) MarginLevelsSubscribe(req *protoapi.MarginLevelsSubscribeRequest, srv protoapi.TradingDataService_MarginLevelsSubscribeServer) error {

	// Wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	marginLevelsChan, ref := t.riskService.ObserveMarginLevels(ctx, t.Config.StreamRetries, req.PartyId, req.MarketId)

	if t.log.GetLevel() == logging.DebugLevel {
		t.log.Debug("Margin levels subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	return observe(ctx, t.log, "MarginLevel", marginLevelsChan, ref, func(ml entities.MarginLevels) error {
		protoMl, err := ml.ToProto(t.accountService)
		if err != nil {
			return apiError(codes.Internal, err)
		}

		return srv.Send(&protoapi.MarginLevelsSubscribeResponse{
			MarginLevels: protoMl,
		})
	})
}

func (t *tradingDataService) EstimateMargin(ctx context.Context, req *protoapi.EstimateMarginRequest) (*protoapi.EstimateMarginResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("EstimateMargin SQL")()
	if req.Order == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("missing order"))
	}

	margin, err := t.estimateMargin(ctx, req.Order)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.EstimateMarginResponse{
		MarginLevels: margin,
	}, nil
}

func (t *tradingDataService) estimateMargin(ctx context.Context, order *vega.Order) (*vega.MarginLevels, error) {
	if order.Side == vega.Side_SIDE_UNSPECIFIED {
		return nil, ErrInvalidOrderSide
	}

	// first get the risk factors and market data (marketdata->markprice)
	rf, err := t.riskFactorService.GetMarketRiskFactors(ctx, order.MarketId)
	if err != nil {
		return nil, err
	}
	mkt, err := t.marketService.GetByID(ctx, order.MarketId)
	if err != nil {
		return nil, err
	}

	mktProto := mkt.ToProto()

	mktData, err := t.marketDataService.GetMarketDataByID(ctx, order.MarketId)
	if err != nil {
		return nil, err
	}

	f, err := num.DecimalFromString(rf.Short.String())
	if err != nil {
		return nil, err
	}
	if order.Side == vega.Side_SIDE_BUY {
		f, err = num.DecimalFromString(rf.Long.String())
		if err != nil {
			return nil, err
		}
	}

	asset, err := mktProto.GetAsset()
	if err != nil {
		return nil, err
	}

	// now calculate margin maintenance
	markPrice, _ := num.DecimalFromString(mktData.MarkPrice.String())

	// if the order is a limit order, use the limit price to calculate the margin maintenance
	if order.Type == vega.Order_TYPE_LIMIT {
		markPrice, _ = num.DecimalFromString(order.Price)
	}

	maintenanceMargin := num.DecimalFromFloat(float64(order.Size)).Mul(f).Mul(markPrice)
	// now we use the risk factors
	return &vega.MarginLevels{
		PartyId:                order.PartyId,
		MarketId:               mktProto.GetId(),
		Asset:                  asset,
		Timestamp:              0,
		MaintenanceMargin:      maintenanceMargin.String(),
		SearchLevel:            maintenanceMargin.Mul(num.DecimalFromFloat(mkt.TradableInstrument.MarginCalculator.ScalingFactors.SearchLevel)).String(),
		InitialMargin:          maintenanceMargin.Mul(num.DecimalFromFloat(mkt.TradableInstrument.MarginCalculator.ScalingFactors.InitialMargin)).String(),
		CollateralReleaseLevel: maintenanceMargin.Mul(num.DecimalFromFloat(mkt.TradableInstrument.MarginCalculator.ScalingFactors.CollateralRelease)).String(),
	}, nil
}

func (t *tradingDataService) EstimateFee(ctx context.Context, req *protoapi.EstimateFeeRequest) (*protoapi.EstimateFeeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("EstimateFee SQL")()
	if req.Order == nil {
		return nil, apiError(codes.InvalidArgument, errors.New("missing order"))
	}

	fee, err := t.estimateFee(ctx, req.Order)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.EstimateFeeResponse{
		Fee: fee,
	}, nil
}

func (t *tradingDataService) estimateFee(ctx context.Context, order *vega.Order) (*vega.Fee, error) {
	mkt, err := t.marketService.GetByID(ctx, order.MarketId)
	if err != nil {
		return nil, err
	}
	price, overflowed := num.UintFromString(order.Price, 10)
	if overflowed {
		return nil, errors.New("invalid order price")
	}
	if order.PeggedOrder != nil {
		return &vega.Fee{
			MakerFee:          "0",
			InfrastructureFee: "0",
			LiquidityFee:      "0",
		}, nil
	}

	base := num.DecimalFromUint(price.Mul(price, num.NewUint(order.Size)))
	maker, infra, liquidity, err := t.feeFactors(mkt)
	if err != nil {
		return nil, err
	}

	fee := &vega.Fee{
		MakerFee:          base.Mul(num.NewDecimalFromFloat(maker)).String(),
		InfrastructureFee: base.Mul(num.NewDecimalFromFloat(infra)).String(),
		LiquidityFee:      base.Mul(num.NewDecimalFromFloat(liquidity)).String(),
	}

	return fee, nil
}

func (t *tradingDataService) feeFactors(mkt entities.Market) (maker, infra, liquidity float64, err error) {
	if maker, err = strconv.ParseFloat(mkt.Fees.Factors.MakerFee, 64); err != nil {
		return
	}
	if infra, err = strconv.ParseFloat(mkt.Fees.Factors.InfrastructureFee, 64); err != nil {
		return
	}
	if liquidity, err = strconv.ParseFloat(mkt.Fees.Factors.LiquidityFee, 64); err != nil {
		return
	}

	return
}

// MarginLevels returns the current margin levels for a given party and market.
func (t *tradingDataService) MarginLevels(ctx context.Context, req *protoapi.MarginLevelsRequest) (*protoapi.MarginLevelsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("MarginLevels SQL")()

	mls, err := t.riskService.GetMarginLevelsByID(ctx, req.PartyId, req.MarketId, entities.OffsetPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, ErrRiskServiceGetMarginLevelsByID, err)
	}
	levels := make([]*vega.MarginLevels, 0, len(mls))
	for _, v := range mls {
		proto, err := v.ToProto(t.accountService)
		if err != nil {
			return nil, apiError(codes.Internal, ErrRiskServiceGetMarginLevelsByID, err)
		}
		levels = append(levels, proto)
	}
	return &protoapi.MarginLevelsResponse{
		MarginLevels: levels,
	}, nil
}

func (t *tradingDataService) GetRiskFactors(ctx context.Context, in *protoapi.GetRiskFactorsRequest) (*protoapi.GetRiskFactorsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetRiskFactors SQL")()

	rfs, err := t.riskFactorService.GetMarketRiskFactors(ctx, in.MarketId)
	if err != nil {
		return nil, nil
	}

	return &protoapi.GetRiskFactorsResponse{
		RiskFactor: rfs.ToProto(),
	}, nil
}

func (t *tradingDataService) Withdrawal(ctx context.Context, req *protoapi.WithdrawalRequest) (*protoapi.WithdrawalResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Withdrawal SQL")()
	if len(req.Id) <= 0 {
		return nil, ErrMissingDepositID
	}
	withdrawal, err := t.withdrawalService.GetByID(ctx, req.Id)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}
	return &protoapi.WithdrawalResponse{
		Withdrawal: withdrawal.ToProto(),
	}, nil
}

func (t *tradingDataService) Withdrawals(ctx context.Context, req *protoapi.WithdrawalsRequest) (*protoapi.WithdrawalsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Withdrawals SQL")()
	if len(req.PartyId) <= 0 {
		return nil, ErrMissingPartyID
	}

	// current API doesn't support pagination, but we will need to support it for v2
	withdrawals, _, err := t.withdrawalService.GetByParty(ctx, req.PartyId, false, entities.OffsetPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}
	out := make([]*vega.Withdrawal, 0, len(withdrawals))
	for _, w := range withdrawals {
		out = append(out, w.ToProto())
	}
	return &protoapi.WithdrawalsResponse{
		Withdrawals: out,
	}, nil
}

func (t *tradingDataService) ERC20WithdrawalApproval(ctx context.Context, req *protoapi.ERC20WithdrawalApprovalRequest) (*protoapi.ERC20WithdrawalApprovalResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ERC20WithdrawalApproval SQL")()
	if len(req.WithdrawalId) <= 0 {
		return nil, ErrMissingDepositID
	}

	// get withdrawal first
	w, err := t.withdrawalService.GetByID(ctx, req.WithdrawalId)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	// get the signatures from  notaryService
	signatures, err := t.notaryService.GetByResourceID(ctx, req.WithdrawalId)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	// some assets stuff
	assets, err := t.assetService.GetAll(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	// get the signature into the form form:
	// 0x + sig1 + sig2 + ... + sigN in hex encoded form
	pack := "0x"
	for _, v := range signatures {
		pack = fmt.Sprintf("%v%v", pack, hex.EncodeToString(v.Sig))
	}

	var address string
	for _, v := range assets {
		if v.ID == w.Asset {
			address = v.ERC20Contract
			break // found the one we want
		}
	}
	if len(address) <= 0 {
		return nil, fmt.Errorf("invalid erc20 token contract address")
	}

	return &protoapi.ERC20WithdrawalApprovalResponse{
		AssetSource:   address,
		Amount:        fmt.Sprintf("%v", w.Amount),
		Expiry:        w.Expiry.UnixMicro(),
		Nonce:         w.Ref,
		TargetAddress: w.Ext.GetErc20().ReceiverAddress,
		Signatures:    pack,
		// timestamps is unix nano, contract needs unix. So load if first, and cut nanos
		Creation: w.CreatedTimestamp.Unix(),
	}, nil
}

func (t *tradingDataService) GetNodeSignaturesAggregate(ctx context.Context,
	req *protoapi.GetNodeSignaturesAggregateRequest,
) (*protoapi.GetNodeSignaturesAggregateResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNodeSignaturesAggregate SQL")()
	if len(req.Id) <= 0 {
		return nil, apiError(codes.InvalidArgument, errors.New("missing ID"))
	}

	sigs, err := t.notaryService.GetByResourceID(ctx, req.Id)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}

	out := make([]*commandspb.NodeSignature, 0, len(sigs))
	for _, v := range sigs {
		vv := v.ToProto()
		out = append(out, vv)
	}

	return &protoapi.GetNodeSignaturesAggregateResponse{
		Signatures: out,
	}, nil
}

func (t *tradingDataService) OracleSpec(ctx context.Context, req *protoapi.OracleSpecRequest) (*protoapi.OracleSpecResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OracleSpec SQL")()
	if len(req.Id) <= 0 {
		return nil, ErrMissingOracleSpecID
	}
	spec, err := t.oracleSpecService.GetSpecByID(ctx, req.Id)
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}
	return &protoapi.OracleSpecResponse{
		OracleSpec: spec.ToProto(),
	}, nil
}

func (t *tradingDataService) OracleSpecs(ctx context.Context, _ *protoapi.OracleSpecsRequest) (*protoapi.OracleSpecsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OracleSpecs SQL")()
	specs, err := t.oracleSpecService.GetSpecs(ctx, entities.OffsetPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	out := make([]*oraclespb.OracleSpec, 0, len(specs))
	for _, v := range specs {
		out = append(out, v.ToProto())
	}

	return &protoapi.OracleSpecsResponse{
		OracleSpecs: out,
	}, nil
}

func (t *tradingDataService) OracleDataBySpec(ctx context.Context, req *protoapi.OracleDataBySpecRequest) (*protoapi.OracleDataBySpecResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("OracleDataBySpec SQL")()
	if len(req.Id) <= 0 {
		return nil, ErrMissingOracleSpecID
	}
	data, _, err := t.oracleDataService.GetOracleDataBySpecID(ctx, req.Id, entities.OffsetPagination{})
	if err != nil {
		return nil, apiError(codes.NotFound, err)
	}
	out := make([]*oraclespb.OracleData, 0, len(data))
	for _, v := range data {
		out = append(out, v.ToProto())
	}
	return &protoapi.OracleDataBySpecResponse{
		OracleData: out,
	}, nil
}

func (t *tradingDataService) ListOracleData(ctx context.Context, _ *protoapi.ListOracleDataRequest) (*protoapi.ListOracleDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("ListOracleData SQL")()
	specs, _, err := t.oracleDataService.ListOracleData(ctx, entities.OffsetPagination{})
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	out := make([]*oraclespb.OracleData, 0, len(specs))
	for _, v := range specs {
		out = append(out, v.ToProto())
	}

	return &protoapi.ListOracleDataResponse{
		OracleData: out,
	}, nil
}

func (t *tradingDataService) LiquidityProvisions(ctx context.Context, req *protoapi.LiquidityProvisionsRequest) (*protoapi.LiquidityProvisionsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("LiquidityProvisions")()

	partyID := entities.NewPartyID(req.Party)
	marketID := entities.NewMarketID(req.Market)

	lps, _, err := t.liquidityProvisionService.Get(ctx, partyID, marketID, "", entities.OffsetPagination{})
	if err != nil {
		return nil, err
	}

	out := make([]*vega.LiquidityProvision, 0, len(lps))
	for _, v := range lps {
		out = append(out, v.ToProto())
	}
	return &protoapi.LiquidityProvisionsResponse{
		LiquidityProvisions: out,
	}, nil
}

func (t *tradingDataService) PartyStake(ctx context.Context, req *protoapi.PartyStakeRequest) (*protoapi.PartyStakeResponse, error) {
	if len(req.Party) <= 0 {
		return nil, apiError(codes.InvalidArgument, errors.New("missing party id"))
	}

	partyID := entities.NewPartyID(req.Party)

	stake, stakeLinkings := t.stakeLinkingService.GetStake(ctx, partyID, entities.OffsetPagination{})
	outStakeLinkings := make([]*eventspb.StakeLinking, 0, len(stakeLinkings))
	for _, v := range stakeLinkings {
		outStakeLinkings = append(outStakeLinkings, v.ToProto())
	}

	return &protoapi.PartyStakeResponse{
		CurrentStakeAvailable: num.UintToString(stake),
		StakeLinkings:         outStakeLinkings,
	}, nil
}

func (t *tradingDataService) GetKeyRotations(ctx context.Context, req *protoapi.GetKeyRotationsRequest) (*protoapi.GetKeyRotationsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetKeyRotations")()

	rotations, err := t.keyRotationService.GetAllPubKeyRotations(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	protoRotations := make([]*protoapi.KeyRotation, len(rotations))
	for i, v := range rotations {
		protoRotations[i] = v.ToProto()
	}

	return &protoapi.GetKeyRotationsResponse{
		Rotations: protoRotations,
	}, nil
}

func (t *tradingDataService) GetKeyRotationsByNode(ctx context.Context, req *protoapi.GetKeyRotationsByNodeRequest) (*protoapi.GetKeyRotationsByNodeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetKeyRotationsByNode")()

	if req.GetNodeId() == "" {
		return nil, apiError(codes.InvalidArgument, errors.New("missing node ID parameter"))
	}

	rotations, err := t.keyRotationService.GetPubKeyRotationsPerNode(ctx, req.GetNodeId())
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	protoRotations := make([]*protoapi.KeyRotation, len(rotations))
	for i, v := range rotations {
		protoRotations[i] = v.ToProto()
	}

	return &protoapi.GetKeyRotationsByNodeResponse{
		Rotations: protoRotations,
	}, nil
}

func (t *tradingDataService) GetNodeData(ctx context.Context, req *protoapi.GetNodeDataRequest) (*protoapi.GetNodeDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNodeData")()

	nodeData, err := t.nodeService.GetNodeData(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.GetNodeDataResponse{
		NodeData: nodeData.ToProto(),
	}, nil
}

func (t *tradingDataService) GetNodes(ctx context.Context, req *protoapi.GetNodesRequest) (*protoapi.GetNodesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNodes")()

	epoch, err := t.epochService.GetCurrent(ctx)
	if err != nil {
		fmt.Printf("%v", err)
		return nil, apiError(codes.Internal, err)
	}

	nodes, err := t.nodeService.GetNodes(ctx, uint64(epoch.ID))
	if err != nil {
		fmt.Printf("%v", err)
		return nil, apiError(codes.Internal, err)
	}

	protoNodes := make([]*vega.Node, len(nodes))
	for i, v := range nodes {
		protoNodes[i] = v.ToProto()
	}

	return &protoapi.GetNodesResponse{
		Nodes: protoNodes,
	}, nil
}

func (t *tradingDataService) GetNodeByID(ctx context.Context, req *protoapi.GetNodeByIDRequest) (*protoapi.GetNodeByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNodeByID")()

	if req.GetId() == "" {
		return nil, apiError(codes.InvalidArgument, errors.New("missing node ID parameter"))
	}

	epoch, err := t.epochService.GetCurrent(ctx)
	if err != nil {
		fmt.Printf("%v", err)
		return nil, apiError(codes.Internal, err)
	}

	node, err := t.nodeService.GetNodeByID(ctx, req.GetId(), uint64(epoch.ID))
	if err != nil {
		fmt.Printf("%v", err)
		return nil, apiError(codes.NotFound, err)
	}

	return &protoapi.GetNodeByIDResponse{
		Node: node.ToProto(),
	}, nil
}

type tradingDataEventBusServer struct {
	stream protoapi.TradingDataService_ObserveEventBusServer
}

func (t tradingDataEventBusServer) RecvMsg(m interface{}) error {
	return t.stream.RecvMsg(m)
}

func (t tradingDataEventBusServer) Context() context.Context {
	return t.stream.Context()
}

func (t tradingDataEventBusServer) Send(data []*eventspb.BusEvent) error {
	resp := &protoapi.ObserveEventBusResponse{
		Events: data,
	}
	return t.stream.Send(resp)
}

func (t *tradingDataService) ObserveEventBus(
	stream protoapi.TradingDataService_ObserveEventBusServer) error {

	server := tradingDataEventBusServer{stream}
	eventService := t.eventService

	return observeEventBus(t.log, t.Config, server, eventService)
}
