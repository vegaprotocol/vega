package monitoring

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/logging"

	types "code.vegaprotocol.io/vega/proto"
)

// Status holds a collection of monitoring services, for checking the state of internal or external resources.
type Status struct {
	log        *logging.Logger
	Blockchain *ChainStatus
}

// ChainStatus provides the current status of the underlying blockchain provider, given a blockchain.Client.
type ChainStatus struct {
	log      *logging.Logger
	interval time.Duration
	client   blockchain.Client
	clientMu sync.Mutex
	status   uint32
	cancel   func()
}

// NewStatusChecker creates a Status checker, currently this is limited to the underlying blockchain status, but
// will be expanded over time to include other services. Once created, a go-routine will start up and
// immediately begin checking at an interval, currently defined internally and set to every 500 milliseconds.
func NewStatusChecker(log *logging.Logger, clt blockchain.Client, interval time.Duration) *Status {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Status{
		log: log,
		Blockchain: &ChainStatus{
			interval: interval, // 500 * time.Millisecond,
			client:   clt,
			clientMu: sync.Mutex{},
			status:   uint32(types.AppStatus_DISCONNECTED),
			cancel:   cancel,
			log:      log,
		},
	}
	go s.Blockchain.start(ctx)
	return s
}

// Stop the internal checker(s) from periodically calling their underlying providers
// Note: currently the only way to start checking externally is to New up a new Status checker
func (s *Status) Stop() {
	s.Blockchain.Stop()
}

// Status returns the current status of the underlying Blockchain connection.
// Returned states are currently CONNECTED, REPLAYING or DISCONNECTED.
func (cs *ChainStatus) Status() types.AppStatus {
	return types.AppStatus(atomic.LoadUint32(&cs.status))
}

func (cs *ChainStatus) SetClient(clt blockchain.Client) {
	cs.clientMu.Lock()
	defer cs.clientMu.Unlock()

	cs.client = clt
}

func (cs *ChainStatus) setStatus(status types.AppStatus) {
	atomic.StoreUint32(&cs.status, uint32(status))
}

func (cs *ChainStatus) tick(status types.AppStatus) types.AppStatus {
	cs.clientMu.Lock()
	defer cs.clientMu.Unlock()
	newStatus := status
	_, err := cs.client.Health()
	if status == types.AppStatus_DISCONNECTED && err == nil {
		// node is connected, now let's check if we are replaying
		res, err2 := cs.client.GetStatus(context.Background())
		if err2 != nil {
			// error while getting the status, maybe we are not
			// really connected, let's not change the status then
			return status
		}
		if res.SyncInfo.CatchingUp {
			newStatus = types.AppStatus_CHAIN_REPLAYING
		} else {
			newStatus = types.AppStatus_CONNECTED
		}
	} else if status == types.AppStatus_CONNECTED && err != nil {
		newStatus = types.AppStatus_DISCONNECTED
	}

	if status == newStatus {
		return status
	}

	cs.setStatus(newStatus)
	cs.log.Debug("Blockchain status updated",
		logging.String("status-old", status.String()),
		logging.String("status-new", newStatus.String()))
	return newStatus
}

// Calling start ideally by go-routine will start periodic checking at interval specified by config
// until context cancel is triggered, typically by the stop function call.
func (cs *ChainStatus) start(ctx context.Context) {
	ticker := time.NewTicker(cs.interval)
	currentStatus := cs.Status()
	for {
		select {
		case <-ticker.C:
			currentStatus = cs.tick(currentStatus)

		case <-ctx.Done():
			return
		}
	}
}

// Stop the internal checker from periodically calling the underlying blockchain provider
// Note: currently the only way to start checking externally is to New up a new Status checker
func (cs *ChainStatus) Stop() {
	cs.cancel()
}
