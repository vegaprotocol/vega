package api

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/go-wallet/crypto"
	protoapi "code.vegaprotocol.io/protos/vega/api"
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

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc/codes"
)

var (
	ErrInvalidSignature           = errors.New("invalid signature")
	ErrSubmitTxCommitDisabled     = errors.New("broadcast_tx_commit is disabled")
	ErrUnknownSubmitTxRequestType = errors.New("invalid broadcast_tx type")
)

type tradingService struct {
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

// no need for a mutext - we only access the config through a value receiver
func (t *tradingService) updateConfig(conf Config) {
	t.conf = conf
}

func (t *tradingService) LastBlockHeight(
	ctx context.Context,
	req *protoapi.LastBlockHeightRequest,
) (*protoapi.LastBlockHeightResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("LastBlockHeight")()
	return &protoapi.LastBlockHeightResponse{
		Height: t.stats.Blockchain.Height(),
	}, nil
}

// GetVegaTime returns the latest blockchain header timestamp, in UnixNano format.
// Example: "1568025900111222333" corresponds to 2019-09-09T10:45:00.111222333Z.
func (t *tradingService) GetVegaTime(ctx context.Context, _ *protoapi.GetVegaTimeRequest) (*protoapi.GetVegaTimeResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetVegaTime")()
	return &protoapi.GetVegaTimeResponse{
		Timestamp: t.timesvc.GetTimeNow().UnixNano(),
	}, nil
}

func (t *tradingService) SubmitTransaction(ctx context.Context, req *protoapi.SubmitTransactionRequest) (*protoapi.SubmitTransactionResponse, error) {
	startTime := time.Now()
	defer metrics.APIRequestAndTimeGRPC("SubmitTransactionV2", startTime)

	if req == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	if err := t.blockchain.SubmitTransactionV2(ctx, req.Tx, protoapi.SubmitTransactionRequest_TYPE_ASYNC); err != nil {
		// This is Tendermint's specific error signature
		if _, ok := err.(interface {
			Code() uint32
			Details() string
			Error() string
		}); ok {
			t.log.Debug("unable to submit transaction", logging.Error(err))
			return nil, apiError(codes.InvalidArgument, err)
		}
		t.log.Debug("unable to submit transaction", logging.Error(err))
		return nil, apiError(codes.Internal, err)
	}

	return &protoapi.SubmitTransactionResponse{
		Success: true,
	}, nil
}

func (t *tradingService) PropagateChainEvent(ctx context.Context, req *protoapi.PropagateChainEventRequest) (*protoapi.PropagateChainEventResponse, error) {
	if req.Event == nil {
		return nil, apiError(codes.InvalidArgument, ErrMalformedRequest)
	}

	// verify the signature then
	err := verifySignature(t.log, req.Event, req.Signature, req.PubKey)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("not a valid signature: %w", err))
	}

	evt := commandspb.ChainEvent{}
	err = proto.Unmarshal(req.Event, &evt)
	if err != nil {
		return nil, apiError(codes.InvalidArgument, fmt.Errorf("not a valid chain event: %w", err))
	}

	var ok = true
	err = t.evtForwarder.Forward(ctx, &evt, req.PubKey)
	if err != nil && err != evtforward.ErrEvtAlreadyExist {
		t.log.Error("unable to forward chain event",
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
// Vega counts for parties, markets, order actions (amend, cancel, submit), Vega version
func (t *tradingService) Statistics(ctx context.Context, _ *protoapi.StatisticsRequest) (*protoapi.StatisticsResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("Statistics")()
	// Call tendermint and related services to get information for statistics
	// We load read-only internal statistics through each package level statistics structs
	epochTime := t.timesvc.GetTimeNow()

	// Call tendermint via rpc client
	var (
		backlogLength, numPeers int
		gt                      *time.Time
		chainID                 string
	)

	backlogLength, numPeers, gt, chainID, err := t.getTendermintStats(ctx)
	if err != nil {
		// do not return an error, let just eventually log it
		t.log.Debug("could not load tendermint stats", logging.Error(err))
	}

	// If the chain is replaying then genesis time can be nil
	genesisTime := ""
	if gt != nil {
		genesisTime = vegatime.Format(*gt)
	}

	stats := &protoapi.Statistics{
		BlockHeight:           t.stats.Blockchain.Height(),
		BacklogLength:         uint64(backlogLength),
		TotalPeers:            uint64(numPeers),
		GenesisTime:           genesisTime,
		CurrentTime:           vegatime.Format(vegatime.Now()),
		VegaTime:              vegatime.Format(epochTime),
		Uptime:                vegatime.Format(t.stats.GetUptime()),
		TxPerBlock:            t.stats.Blockchain.TotalTxLastBatch(),
		AverageTxBytes:        t.stats.Blockchain.AverageTxSizeBytes(),
		AverageOrdersPerBlock: t.stats.Blockchain.AverageOrdersPerBatch(),
		TradesPerSecond:       t.stats.Blockchain.TradesPerSecond(),
		OrdersPerSecond:       t.stats.Blockchain.OrdersPerSecond(),
		Status:                t.statusChecker.ChainStatus(),
		AppVersionHash:        t.stats.GetVersionHash(),
		AppVersion:            t.stats.GetVersion(),
		ChainVersion:          t.stats.GetChainVersion(),
		TotalAmendOrder:       t.stats.Blockchain.TotalAmendOrder(),
		TotalCancelOrder:      t.stats.Blockchain.TotalCancelOrder(),
		TotalCreateOrder:      t.stats.Blockchain.TotalCreateOrder(),
		TotalOrders:           t.stats.Blockchain.TotalOrders(),
		TotalTrades:           t.stats.Blockchain.TotalTrades(),
		BlockDuration:         t.stats.Blockchain.BlockDuration(),
		ChainId:               chainID,
	}
	return &protoapi.StatisticsResponse{
		Statistics: stats,
	}, nil
}

func (t *tradingService) getTendermintStats(
	ctx context.Context,
) (
	backlogLength, numPeers int,
	genesis *time.Time,
	chainID string,
	err error,
) {

	if t.stats == nil || t.stats.Blockchain == nil {
		return 0, 0, nil, "", apiError(codes.Internal, ErrChainNotConnected)
	}

	const refused = "connection refused"

	// Unconfirmed TX count == current transaction backlog length
	backlogLength, err = t.blockchain.GetUnconfirmedTxCount(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return 0, 0, nil, "", nil
		}
		return 0, 0, nil, "", apiError(codes.Internal, ErrBlockchainBacklogLength, err)
	}

	if atomic.LoadUint32(&t.hasGenesisTimeAndChainID) == 0 {
		if err = t.getGenesisTimeAndChainID(ctx); err != nil {
			return 0, 0, nil, "", err
		}
	}

	// Net info provides peer stats etc (block chain network info) == number of peers
	netInfo, err := t.getTMNetInfo(ctx)
	if err != nil {
		return backlogLength, 0, &t.genesisTime, t.chainID, nil
	}

	return backlogLength, netInfo.NPeers, &t.genesisTime, t.chainID, nil
}

func (t *tradingService) getTMNetInfo(ctx context.Context) (tmctypes.ResultNetInfo, error) {
	t.netInfoMu.RLock()
	defer t.netInfoMu.RUnlock()

	if t.netInfo == nil {
		return tmctypes.ResultNetInfo{}, apiError(codes.Internal, ErrBlockchainNetworkInfo)
	}

	return *t.netInfo, nil
}

func (t *tradingService) updateNetInfo(ctx context.Context) {
	// update the net info every 1 minutes
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			netInfo, err := t.blockchain.GetNetworkInfo(ctx)
			if err != nil {
				continue
			}
			t.netInfoMu.Lock()
			t.netInfo = netInfo
			t.netInfoMu.Unlock()
		}
	}
}

func (t *tradingService) getGenesisTimeAndChainID(ctx context.Context) error {
	const refused = "connection refused"
	// just lock in here, ideally we'll come here only once, so not a big issue to lock
	t.mu.Lock()
	defer t.mu.Unlock()

	var err error
	// Genesis retrieves the current genesis date/time for the blockchain
	t.genesisTime, err = t.blockchain.GetGenesisTime(ctx)
	if err != nil {
		if strings.Contains(err.Error(), refused) {
			return nil
		}
		return apiError(codes.Internal, ErrBlockchainGenesisTime, err)
	}

	t.chainID, err = t.blockchain.GetChainID(ctx)
	if err != nil {
		return apiError(codes.Internal, ErrBlockchainChainID, err)
	}

	atomic.StoreUint32(&t.hasGenesisTimeAndChainID, 1)
	return nil
}

func (t *tradingService) ObserveEventBus(
	stream protoapi.TradingService_ObserveEventBusServer) error {
	defer metrics.StartAPIRequestAndTimeGRPC("ObserveEventBus")()

	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()

	// now we start listening for a few seconds in order to get at least the very first message
	// this will be blocking until the connection by the client is closed
	// and we will not start processing any events until we receive the original request
	// indicating filters and batch size.
	req, err := t.recvEventRequest(stream)
	if err != nil {
		// client exited, nothing to do
		return nil
	}

	if err := req.Validate(); err != nil {
		return apiError(codes.InvalidArgument, ErrMalformedRequest, err)
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
	ch, bCh := t.eventService.ObserveEvents(ctx, t.conf.StreamRetries, types, int(req.BatchSize), filters...)
	defer close(bCh)

	if req.BatchSize > 0 {
		err := t.observeEventsWithAck(ctx, stream, req.BatchSize, ch, bCh)
		return err

	}
	err = t.observeEvents(ctx, stream, ch)
	return err
}

func (t *tradingService) observeEvents(
	ctx context.Context,
	stream protoapi.TradingService_ObserveEventBusServer,
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
				t.log.Error("Error sending event on stream", logging.Error(err))
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		}
	}
}

func (t *tradingService) recvEventRequest(
	stream protoapi.TradingService_ObserveEventBusServer,
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

func (t *tradingService) observeEventsWithAck(
	ctx context.Context,
	stream protoapi.TradingService_ObserveEventBusServer,
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
				t.log.Error("Error sending event on stream", logging.Error(err))
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		}

		// now we try to read again the new size / ack
		req, err := t.recvEventRequest(stream)
		if err != nil {
			return err
		}

		if req.BatchSize != batchSize {
			batchSize = req.BatchSize
			bCh <- int(batchSize)
		}
	}
}
