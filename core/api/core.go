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

package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/evtforward"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/subscribers"
	"code.vegaprotocol.io/vega/core/vegatime"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	ptypes "code.vegaprotocol.io/vega/protos/vega"
	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/pkg/errors"
	"github.com/tendermint/tendermint/libs/bytes"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc/codes"
)

var (
	ErrInvalidSignature           = errors.New("invalid signature")
	ErrSubmitTxCommitDisabled     = errors.New("broadcast_tx_commit is disabled")
	ErrUnknownSubmitTxRequestType = errors.New("invalid broadcast_tx type")
)

type coreService struct {
	protoapi.UnimplementedCoreServiceServer
	log  *logging.Logger
	conf Config

	blockchain Blockchain
	stats      *stats.Stats

	svcMu        sync.RWMutex
	evtForwarder EvtForwarder
	timesvc      TimeService
	eventService EventService
	subCancels   []func()
	powParams    ProofOfWorkParams
	spamEngine   SpamEngine
	powEngine    PowEngine

	chainID                  string
	genesisTime              time.Time
	hasGenesisTimeAndChainID atomic.Bool
	mu                       sync.Mutex

	netInfo   *tmctypes.ResultNetInfo
	netInfoMu sync.RWMutex
}

func (s *coreService) UpdateProtocolServices(
	evtforwarder EvtForwarder,
	timesvc TimeService,
	evtsvc EventService,
	powParams ProofOfWorkParams,
) {
	s.svcMu.Lock()
	defer s.svcMu.Unlock()
	// first cancel all subscriptions
	for _, f := range s.subCancels {
		f()
	}
	s.evtForwarder = evtforwarder
	s.eventService = evtsvc
	s.timesvc = timesvc
	s.powParams = powParams
}

// no need for a mutex - we only access the config through a value receiver.
func (s *coreService) updateConfig(conf Config) {
	s.conf = conf
}

func (s *coreService) LastBlockHeight(
	ctx context.Context,
	req *protoapi.LastBlockHeightRequest,
) (*protoapi.LastBlockHeightResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("LastBlockHeight")()

	if !s.hasGenesisTimeAndChainID.Load() {
		if err := s.getGenesisTimeAndChainID(ctx); err != nil {
			return nil, fmt.Errorf("failed to intialise chainID: %w", err)
		}
	}

	blockHeight, blockHash := s.powParams.BlockData()
	if s.log.IsDebug() {
		s.log.Debug("block height requested, returning", logging.Uint64("block-height", blockHeight), logging.String("block hash", blockHash), logging.String("chaindID", s.chainID))
	}

	if !s.powParams.IsReady() {
		return nil, errors.New("Failed to get last block height server is initialising")
	}

	return &protoapi.LastBlockHeightResponse{
		Height:                      blockHeight,
		Hash:                        blockHash,
		SpamPowDifficulty:           s.powParams.SpamPoWDifficulty(),
		SpamPowHashFunction:         s.powParams.SpamPoWHashFunction(),
		SpamPowNumberOfPastBlocks:   s.powParams.SpamPoWNumberOfPastBlocks(),
		SpamPowNumberOfTxPerBlock:   s.powParams.SpamPoWNumberOfTxPerBlock(),
		SpamPowIncreasingDifficulty: s.powParams.SpamPoWIncreasingDifficulty(),
		ChainId:                     s.chainID,
	}, nil
}

// GetVegaTime returns the latest blockchain header timestamp, in UnixNano format.
// Example: "1568025900111222333" corresponds to 2019-09-09T10:45:00.111222333Z.
func (s *coreService) GetVegaTime(ctx context.Context, _ *protoapi.GetVegaTimeRequest) (*protoapi.GetVegaTimeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetVegaTime")()
	s.svcMu.RLock()
	defer s.svcMu.RUnlock()

	return &protoapi.GetVegaTimeResponse{
		Timestamp: s.timesvc.GetTimeNow().UnixNano(),
	}, nil
}

func (s *coreService) SubmitTransaction(ctx context.Context, req *protoapi.SubmitTransactionRequest) (*protoapi.SubmitTransactionResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("SubmitTransaction", startTime)

	if req == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	txResult, err := s.blockchain.SubmitTransactionSync(ctx, req.Tx)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.SubmitTransactionResponse{
		Success: txResult.Code == 0,
		Code:    txResult.Code,
		Data:    string(txResult.Data.Bytes()),
		Log:     txResult.Log,
		Height:  0,
		TxHash:  txResult.Hash.String(),
	}, nil
}

func (s *coreService) CheckTransaction(ctx context.Context, req *protoapi.CheckTransactionRequest) (*protoapi.CheckTransactionResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("CheckTransaction", startTime)

	if req == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	checkResult, err := s.blockchain.CheckTransaction(ctx, req.Tx)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.CheckTransactionResponse{
		Code:      checkResult.Code,
		Data:      string(checkResult.Data),
		Info:      checkResult.Info,
		Log:       checkResult.Log,
		Success:   checkResult.IsOK(),
		GasWanted: checkResult.GasWanted,
		GasUsed:   checkResult.GasUsed,
	}, nil
}

func (s *coreService) CheckRawTransaction(ctx context.Context, req *protoapi.CheckRawTransactionRequest) (*protoapi.CheckRawTransactionResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("CheckRawTransaction", startTime)

	if req == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	checkResult, err := s.blockchain.CheckRawTransaction(ctx, req.Tx)
	if err != nil {
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.CheckRawTransactionResponse{
		Code:      checkResult.Code,
		Data:      string(checkResult.Data),
		Info:      checkResult.Info,
		Log:       checkResult.Log,
		Success:   checkResult.IsOK(),
		GasWanted: checkResult.GasWanted,
		GasUsed:   checkResult.GasUsed,
	}, nil
}

func (s *coreService) PropagateChainEvent(ctx context.Context, req *protoapi.PropagateChainEventRequest) (*protoapi.PropagateChainEventResponse, error) {
	if req.Event == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	// verify the signature then
	err := verifySignature(s.log, req.Event, req.Signature, req.PubKey)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("not a valid signature: %w", err))
	}

	evt := commandspb.ChainEvent{}
	err = proto.Unmarshal(req.Event, &evt)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("not a valid chain event: %w", err))
	}

	ok := true
	s.svcMu.RLock()
	defer s.svcMu.RUnlock()
	err = s.evtForwarder.Forward(ctx, &evt, req.PubKey)
	if err != nil && err != evtforward.ErrEvtAlreadyExist {
		s.log.Error("unable to forward chain event",
			logging.String("pubkey", req.PubKey),
			logging.Error(err))
		if err == evtforward.ErrPubKeyNotAllowlisted {
			return nil, apiError(codes.PermissionDenied, err)
		}
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.PropagateChainEventResponse{
		Success: ok,
	}, nil
}

func verifySignature(
	log *logging.Logger,
	message []byte,
	sig []byte,
	pubKey string,
) error {
	validator, err := crypto.NewSignatureAlgorithm(crypto.Ed25519, 1)
	if err != nil {
		if log != nil {
			log.Error("unable to instantiate new algorithm", logging.Error(err))
		}
		return err
	}

	pubKeyBytes, err := hex.DecodeString(pubKey)
	if err != nil {
		if log != nil {
			log.Error("unable to decode hexencoded ubkey", logging.Error(err))
		}
		return err
	}
	ok, err := validator.Verify(pubKeyBytes, message, sig)
	if err != nil {
		if log != nil {
			log.Error("unable to verify bundle", logging.Error(err))
		}
		return err
	}
	if !ok {
		return ErrInvalidSignature
	}
	return nil
}

// Statistics provides various blockchain and Vega statistics, including:
// Blockchain height, backlog length, current time, orders and trades per block, tendermint version
// Vega counts for parties, markets, order actions (amend, cancel, submit), Vega version.
func (s *coreService) Statistics(ctx context.Context, _ *protoapi.StatisticsRequest) (*protoapi.StatisticsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Statistics")()
	// Call tendermint and related services to get information for statistics
	// We load read-only internal statistics through each package level statistics structs
	s.svcMu.RLock()
	epochTime := s.timesvc.GetTimeNow()
	s.svcMu.RUnlock()

	// Call tendermint via rpc client
	var (
		backlogLength, numPeers int
		gt                      *time.Time
		chainID                 string
	)

	backlogLength, numPeers, gt, chainID, err := s.getTendermintStats(ctx)
	if err != nil {
		// do not return an error, let just eventually log it
		s.log.Debug("could not load tendermint stats", logging.Error(err))
	}

	// If the chain is replaying then genesis time can be nil
	genesisTime := ""
	if gt != nil {
		genesisTime = vegatime.Format(*gt)
	}

	stats := &protoapi.Statistics{
		BlockHeight:           s.stats.Blockchain.Height(),
		BlockHash:             s.stats.Blockchain.Hash(),
		BacklogLength:         uint64(backlogLength),
		TotalPeers:            uint64(numPeers),
		GenesisTime:           genesisTime,
		CurrentTime:           vegatime.Format(time.Now()),
		VegaTime:              vegatime.Format(epochTime),
		Uptime:                vegatime.Format(s.stats.GetUptime()),
		TxPerBlock:            s.stats.Blockchain.TotalTxLastBatch(),
		AverageTxBytes:        s.stats.Blockchain.AverageTxSizeBytes(),
		AverageOrdersPerBlock: s.stats.Blockchain.AverageOrdersPerBatch(),
		TradesPerSecond:       s.stats.Blockchain.TradesPerSecond(),
		OrdersPerSecond:       s.stats.Blockchain.OrdersPerSecond(),
		Status:                ptypes.ChainStatus_CHAIN_STATUS_CONNECTED,
		AppVersionHash:        s.stats.GetVersionHash(),
		AppVersion:            s.stats.GetVersion(),
		ChainVersion:          s.stats.GetChainVersion(),
		TotalAmendOrder:       s.stats.Blockchain.TotalAmendOrder(),
		TotalCancelOrder:      s.stats.Blockchain.TotalCancelOrder(),
		TotalCreateOrder:      s.stats.Blockchain.TotalCreateOrder(),
		TotalOrders:           s.stats.Blockchain.TotalOrders(),
		TotalTrades:           s.stats.Blockchain.TotalTrades(),
		BlockDuration:         s.stats.Blockchain.BlockDuration(),
		EventCount:            s.stats.Blockchain.TotalEventsLastBatch(),
		EventsPerSecond:       s.stats.Blockchain.EventsPerSecond(),
		EpochSeq:              s.stats.GetEpochSeq(),
		EpochStartTime:        vegatime.Format(s.stats.GetEpochStartTime()),
		EpochExpiryTime:       vegatime.Format(s.stats.GetEpochExpireTime()),
		ChainId:               chainID,
	}
	return &protoapi.StatisticsResponse{
		Statistics: stats,
	}, nil
}

func (s *coreService) getTendermintStats(
	ctx context.Context,
) (
	backlogLength, numPeers int,
	genesis *time.Time,
	chainID string,
	err error,
) {
	if s.stats == nil || s.stats.Blockchain == nil {
		return 0, 0, nil, "", apiError(codes.Internal, ErrChainNotConnected)
	}

	const refused = "connection refused"

	// Unconfirmed TX count == current transaction backlog length
	backlogLength, err = s.blockchain.GetUnconfirmedTxCount(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return 0, 0, nil, "", nil
		}
		return 0, 0, nil, "", apiError(codes.Internal, ErrBlockchainBacklogLength, err)
	}

	if !s.hasGenesisTimeAndChainID.Load() {
		if err = s.getGenesisTimeAndChainID(ctx); err != nil {
			return 0, 0, nil, "", err
		}
	}

	// Net info provides peer stats etc (block chain network info) == number of peers
	netInfo, err := s.getTMNetInfo(ctx)
	if err != nil {
		return backlogLength, 0, &s.genesisTime, s.chainID, nil // nolint
	}

	return backlogLength, netInfo.NPeers, &s.genesisTime, s.chainID, nil
}

func (s *coreService) getTMNetInfo(_ context.Context) (tmctypes.ResultNetInfo, error) {
	s.netInfoMu.RLock()
	defer s.netInfoMu.RUnlock()

	if s.netInfo == nil {
		return tmctypes.ResultNetInfo{}, apiError(codes.Internal, ErrBlockchainNetworkInfo)
	}

	return *s.netInfo, nil
}

func (s *coreService) updateNetInfo(ctx context.Context) {
	// update the net info every 1 minutes
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			netInfo, err := s.blockchain.GetNetworkInfo(ctx)
			if err != nil {
				continue
			}
			s.netInfoMu.Lock()
			s.netInfo = netInfo
			s.netInfoMu.Unlock()
		}
	}
}

func (s *coreService) getGenesisTimeAndChainID(ctx context.Context) error {
	const refused = "connection refused"
	// just lock in here, ideally we'll come here only once, so not a big issue to lock
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	// Genesis retrieves the current genesis date/time for the blockchain
	s.genesisTime, err = s.blockchain.GetGenesisTime(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return nil
		}
		return apiError(codes.Internal, ErrBlockchainGenesisTime, err)
	}

	s.chainID, err = s.blockchain.GetChainID(ctx)
	if err != nil {
		return apiError(codes.Internal, ErrBlockchainChainID, err)
	}

	s.hasGenesisTimeAndChainID.Store(true)
	return nil
}

func (s *coreService) ObserveEventBus(
	stream protoapi.CoreService_ObserveEventBusServer,
) error {
	defer metrics.StartAPIRequestAndTimeGRPC("ObserveEventBus")()

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()

	// now we start listening for a few seconds in order to get at least the very first message
	// this will be blocking until the connection by the client is closed
	// and we will not start processing any events until we receive the original request
	// indicating filters and batch size.
	req, err := s.recvEventRequest(stream)
	if err != nil {
		// client exited, nothing to do
		return nil // nolint
	}

	// now we will aggregate filter out of the initial request
	types, err := events.ProtoToInternal(req.Type...)
	if err != nil {
		return apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	filters := []subscribers.EventFilter{}
	if len(req.MarketId) > 0 && len(req.PartyId) > 0 {
		filters = append(filters, events.GetPartyAndMarketFilter(req.MarketId, req.PartyId))
	} else {
		if len(req.MarketId) > 0 {
			filters = append(filters, events.GetMarketIDFilter(req.MarketId))
		}
		if len(req.PartyId) > 0 {
			filters = append(filters, events.GetPartyIDFilter(req.PartyId))
		}
	}

	// here we add the cancel to the list of observer
	// so if a protocol upgrade happen we can stop processing those
	// and nicely do the upgrade
	s.svcMu.Lock()
	s.subCancels = append(s.subCancels, cfunc)
	s.svcMu.Unlock()

	// number of retries to -1 to have pretty much unlimited retries
	ch, bCh := s.eventService.ObserveEvents(ctx, s.conf.StreamRetries, types, int(req.BatchSize), filters...)
	defer close(bCh)

	if req.BatchSize > 0 {
		err := s.observeEventsWithAck(ctx, stream, req.BatchSize, ch, bCh)
		return err
	}
	err = s.observeEvents(ctx, stream, ch)
	return err
}

func (s *coreService) observeEvents(
	ctx context.Context,
	stream protoapi.CoreService_ObserveEventBusServer,
	ch <-chan []*eventspb.BusEvent,
) error {
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return nil
			}
			resp := &protoapi.ObserveEventBusResponse{
				Events: data,
			}
			if err := stream.Send(resp); err != nil {
				s.log.Error("Error sending event on stream", logging.Error(err))
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		}
	}
}

func (s *coreService) recvEventRequest(
	stream protoapi.CoreService_ObserveEventBusServer,
) (*protoapi.ObserveEventBusRequest, error) {
	readCtx, cfunc := context.WithTimeout(stream.Context(), 5*time.Second)
	oebCh := make(chan protoapi.ObserveEventBusRequest)
	var err error
	go func() {
		defer close(oebCh)
		nb := protoapi.ObserveEventBusRequest{}
		if err = stream.RecvMsg(&nb); err != nil {
			cfunc()
			return
		}
		oebCh <- nb
	}()
	select {
	case <-readCtx.Done():
		if err != nil {
			// this means the client disconnected
			return nil, err
		}
		// this mean we timedout
		return nil, readCtx.Err()
	case nb := <-oebCh:
		return &nb, nil
	}
}

func (s *coreService) observeEventsWithAck(
	ctx context.Context,
	stream protoapi.CoreService_ObserveEventBusServer,
	batchSize int64,
	ch <-chan []*eventspb.BusEvent,
	bCh chan<- int,
) error {
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return nil
			}
			resp := &protoapi.ObserveEventBusResponse{
				Events: data,
			}
			if err := stream.Send(resp); err != nil {
				s.log.Error("Error sending event on stream", logging.Error(err))
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		}

		// now we try to read again the new size / ack
		req, err := s.recvEventRequest(stream)
		if err != nil {
			return err
		}

		if req.BatchSize != batchSize {
			batchSize = req.BatchSize
			bCh <- int(batchSize)
		}
	}
}

func (s *coreService) handleSubmitRawTxTMError(err error) error {
	// This is Tendermint's specific error signature
	if _, ok := err.(interface {
		Code() uint32
		Details() string
		Error() string
	}); ok {
		s.log.Debug("unable to submit raw transaction", logging.Error(err))
		return apiError(codes.InvalidArgument, err)
	}
	s.log.Debug("unable to submit raw transaction", logging.Error(err))

	return apiError(codes.Internal, err)
}

func setResponseBasisContent(response *protoapi.SubmitRawTransactionResponse, code uint32, log string, data, hash bytes.HexBytes) {
	response.TxHash = hash.String()
	response.Code = code
	response.Data = data.String()
	response.Log = log
}

func (s *coreService) SubmitRawTransaction(ctx context.Context, req *protoapi.SubmitRawTransactionRequest) (*protoapi.SubmitRawTransactionResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("SubmitTransaction", startTime)

	if req == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	successResponse := &protoapi.SubmitRawTransactionResponse{Success: true}
	switch req.Type {
	case protoapi.SubmitRawTransactionRequest_TYPE_ASYNC:
		txResult, err := s.blockchain.SubmitRawTransactionAsync(ctx, req.Tx)
		if err != nil {
			if txResult != nil {
				return &protoapi.SubmitRawTransactionResponse{
					Success: false,
					Code:    txResult.Code,
					Data:    txResult.Data.String(),
					Log:     txResult.Log,
				}, s.handleSubmitRawTxTMError(err)
			}
			return nil, s.handleSubmitRawTxTMError(err)
		}
		setResponseBasisContent(successResponse, txResult.Code, txResult.Log, txResult.Data, txResult.Hash)

	case protoapi.SubmitRawTransactionRequest_TYPE_SYNC:
		txResult, err := s.blockchain.SubmitRawTransactionSync(ctx, req.Tx)
		if err != nil {
			if txResult != nil {
				return &protoapi.SubmitRawTransactionResponse{
					Success: false,
					Code:    txResult.Code,
					Data:    txResult.Data.String(),
					Log:     txResult.Log,
				}, s.handleSubmitRawTxTMError(err)
			}
			return nil, s.handleSubmitRawTxTMError(err)
		}
		setResponseBasisContent(successResponse, txResult.Code, txResult.Log, txResult.Data, txResult.Hash)

	case protoapi.SubmitRawTransactionRequest_TYPE_COMMIT:
		txResult, err := s.blockchain.SubmitRawTransactionCommit(ctx, req.Tx)
		if err != nil {
			if txResult != nil {
				return &protoapi.SubmitRawTransactionResponse{
					Success: false,
					Code:    txResult.DeliverTx.Code,
					Data:    string(txResult.DeliverTx.Data),
					Log:     txResult.DeliverTx.Log,
				}, s.handleSubmitRawTxTMError(err)
			}
			return nil, s.handleSubmitRawTxTMError(err)
		}
		setResponseBasisContent(successResponse, txResult.DeliverTx.Code, txResult.DeliverTx.Log, txResult.DeliverTx.Data, txResult.Hash)
		successResponse.Height = txResult.Height

	default:
		return nil, apiError(codes.InvalidArgument, errors.New("Invalid TX Type"))
	}

	return successResponse, nil
}

func (s *coreService) GetSpamStatistics(_ context.Context, req *protoapi.GetSpamStatisticsRequest) (*protoapi.GetSpamStatisticsResponse, error) {
	if req.PartyId == "" {
		return nil, apiError(codes.InvalidArgument, ErrEmptyMissingPartyID)
	}

	spamStats := &protoapi.SpamStatistics{}
	// Spam engine is not set when NullBlockChain is used
	if s.spamEngine != nil {
		spamStats = s.spamEngine.GetSpamStatistics(req.PartyId)
	}

	// Noop PoW Engine is used for NullBlockChain so this should be safe
	spamStats.Pow = s.powEngine.GetSpamStatistics(req.PartyId)

	resp := &protoapi.GetSpamStatisticsResponse{
		ChainId:    s.chainID,
		Statistics: spamStats,
	}

	return resp, nil
}
