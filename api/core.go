package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	protoapi "code.vegaprotocol.io/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/evtforward"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/vegatime"
	"code.vegaprotocol.io/vegawallet/crypto"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	tmctypes "github.com/tendermint/tendermint/rpc/coretypes"
	"google.golang.org/grpc/codes"
)

var (
	ErrInvalidSignature           = errors.New("invalid signature")
	ErrSubmitTxCommitDisabled     = errors.New("broadcast_tx_commit is disabled")
	ErrUnknownSubmitTxRequestType = errors.New("invalid broadcast_tx type")
)

type coreService struct {
	log  *logging.Logger
	conf Config

	blockchain    Blockchain
	evtForwarder  EvtForwarder
	timesvc       TimeService
	stats         *stats.Stats
	statusChecker *monitoring.Status
	eventService  EventService

	chainID                  string
	genesisTime              time.Time
	hasGenesisTimeAndChainID uint32
	mu                       sync.Mutex

	netInfo   *tmctypes.ResultNetInfo
	netInfoMu sync.RWMutex
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

	return &protoapi.LastBlockHeightResponse{
		Height: s.stats.Blockchain.Height(),
	}, nil
}

// GetVegaTime returns the latest blockchain header timestamp, in UnixNano format.
// Example: "1568025900111222333" corresponds to 2019-09-09T10:45:00.111222333Z.
func (s *coreService) GetVegaTime(ctx context.Context, _ *protoapi.GetVegaTimeRequest) (*protoapi.GetVegaTimeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetVegaTime")()

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

	txHash, err := s.blockchain.SubmitTransaction(ctx, req.Tx, protoapi.SubmitTransactionRequest_TYPE_SYNC)
	if err != nil {
		// This is Tendermint's specific error signature
		if _, ok := err.(interface {
			Code() uint32
			Details() string
			Error() string
		}); ok {
			s.log.Debug("unable to submit transaction", logging.Error(err))
			return nil, apiError(codes.InvalidArgument, err)
		}
		s.log.Debug("unable to submit transaction", logging.Error(err))

		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.SubmitTransactionResponse{
		Success: true,
		TxHash:  txHash,
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
	err = s.evtForwarder.Forward(ctx, &evt, req.PubKey)
	if err != nil && err != evtforward.ErrEvtAlreadyExist {
		s.log.Error("unable to forward chain event",
			logging.String("pubkey", req.PubKey),
			logging.Error(err))
		if err == evtforward.ErrPubKeyNotAllowlisted {
			return nil, apiError(codes.PermissionDenied, err)
		} else {
			return nil, apiError(codes.Internal, err)
		}
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
	epochTime := s.timesvc.GetTimeNow()

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
		Status:                s.statusChecker.ChainStatus(),
		AppVersionHash:        s.stats.GetVersionHash(),
		AppVersion:            s.stats.GetVersion(),
		ChainVersion:          s.stats.GetChainVersion(),
		TotalAmendOrder:       s.stats.Blockchain.TotalAmendOrder(),
		TotalCancelOrder:      s.stats.Blockchain.TotalCancelOrder(),
		TotalCreateOrder:      s.stats.Blockchain.TotalCreateOrder(),
		TotalOrders:           s.stats.Blockchain.TotalOrders(),
		TotalTrades:           s.stats.Blockchain.TotalTrades(),
		BlockDuration:         s.stats.Blockchain.BlockDuration(),
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

	if atomic.LoadUint32(&s.hasGenesisTimeAndChainID) == 0 {
		if err = s.getGenesisTimeAndChainID(ctx); err != nil {
			return 0, 0, nil, "", err
		}
	}

	// Net info provides peer stats etc (block chain network info) == number of peers
	netInfo, err := s.getTMNetInfo(ctx)
	if err != nil {
		return backlogLength, 0, &s.genesisTime, s.chainID, nil //nolint
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

	atomic.StoreUint32(&s.hasGenesisTimeAndChainID, 1)
	return nil
}

func (s *coreService) ObserveEventBus(
	stream protoapi.CoreService_ObserveEventBusServer) error {
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
		return nil //nolint
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

func (s *coreService) SubmitRawTransaction(ctx context.Context, req *protoapi.SubmitRawTransactionRequest) (*protoapi.SubmitRawTransactionResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("SubmitTransaction", startTime)

	if req == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	txHash, err := s.blockchain.SubmitRawTransaction(ctx, req.Tx, req.Type)
	if err != nil {
		// This is Tendermint's specific error signature
		if _, ok := err.(interface {
			Code() uint32
			Details() string
			Error() string
		}); ok {
			s.log.Debug("unable to submit raw transaction", logging.Error(err))
			return nil, apiError(codes.InvalidArgument, err)
		}
		s.log.Debug("unable to submit raw transaction", logging.Error(err))

		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.SubmitRawTransactionResponse{
		Success: true,
		TxHash:  txHash,
	}, nil
}
