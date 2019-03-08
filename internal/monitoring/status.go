package monitoring

import (
	"context"
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
	log         *logging.Logger
	interval    time.Duration
	client      blockchain.Client
	status      uint32
	cancel      func()
}

// NewStatusChecker creates a Status checker, currently this is limited to the underlying blockchain status, but
// will be expanded over time to include other services. Once created, a go-routine will start up and
// immediately begin checking at an interval, currently defined internally and set to every 500 milliseconds.
func NewStatusChecker(log *logging.Logger, clt blockchain.Client) *Status {
	ctx, cancel := context.WithCancel(context.Background())
		s := &Status{
		log:    log,
		Blockchain: &ChainStatus{
			interval: 500 * time.Millisecond,
			client: clt,
			status: uint32(types.ChainStatus_DISCONNECTED),
			cancel: cancel,
			log:    log,
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
func (cs *ChainStatus) Status() types.ChainStatus {
	return types.ChainStatus(atomic.LoadUint32(&cs.status))
}

func (cs *ChainStatus) setStatus(status types.ChainStatus) {
	atomic.StoreUint32(&cs.status, uint32(status))
}

// Calling start ideally by go-routine will start periodic checking at interval specified by config
// until context cancel is triggered, typically by the stop function call.
func (cs *ChainStatus) start(ctx context.Context) {
	ticker := time.NewTicker(cs.interval)
	currentStatus := cs.Status()
	for {
		select {
		case <-ticker.C:
			oldStatus := currentStatus
			_, err := cs.client.Health()
			if currentStatus == types.ChainStatus_DISCONNECTED && err == nil {
				// node is connected, now let's check if we are replaying
				res, err := cs.client.GetStatus(context.Background())
				if err != nil {
					// error while getting the status, maybe we are not
					// really connected, let's not change the status then
					continue
				}
				if res.SyncInfo.CatchingUp {
					currentStatus = types.ChainStatus_REPLAYING
					cs.setStatus(types.ChainStatus_REPLAYING)
					continue
				}
				currentStatus = types.ChainStatus_CONNECTED
				cs.setStatus(types.ChainStatus_CONNECTED)
			} else if currentStatus == types.ChainStatus_CONNECTED && err != nil {
				currentStatus = types.ChainStatus_DISCONNECTED
				cs.setStatus(types.ChainStatus_DISCONNECTED)
			} else {
				// no status changes
				continue
			}

			cs.log.Debug("Blockchain status updated",
				logging.String("status-old", oldStatus.String()),
				logging.String("status-new", currentStatus.String()))

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
