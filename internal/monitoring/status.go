package monitoring

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"

	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_client_mock.go -package mocks code.vegaprotocol.io/vega/internal/monitoring BlockchainClient
type BlockchainClient interface {
	Health() (*tmctypes.ResultHealth, error)
	GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error)
}

// Status holds a collection of monitoring services, for checking the state of internal or external resources.
type Status struct {
	Config
	log        *logging.Logger
	blockchain *ChainStatus
}

// ChainStatus provides the current status of the underlying blockchain provider, given a blockchain.Client.
type ChainStatus struct {
	log *logging.Logger

	config   Config
	client   BlockchainClient
	status   uint32
	starting bool

	retriesCount int
	cfgMu        sync.Mutex

	cancel            func()
	onChainDisconnect func()
}

// New creates a Status checker, currently this is limited to the underlying blockchain status, but
// will be expanded over time to include other services. Once created, a go-routine will start up and
// immediately begin checking at an interval, currently defined internally and set to every 500 milliseconds.
func New(log *logging.Logger, conf Config, clt BlockchainClient) *Status {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(conf.Level.Get())

	ctx, cancel := context.WithCancel(context.Background())
	s := &Status{
		log:    log,
		Config: conf,
		blockchain: &ChainStatus{
			log:               log,
			config:            conf,
			client:            clt,
			status:            uint32(types.ChainStatus_DISCONNECTED),
			starting:          true,
			retriesCount:      int(conf.Retries),
			cancel:            cancel,
			onChainDisconnect: nil,
		},
	}
	go s.blockchain.start(ctx)
	return s
}

func (s *Status) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.blockchain.cfgMu.Lock()
	s.blockchain.config = cfg
	s.blockchain.retriesCount = int(cfg.Retries)
	s.blockchain.cfgMu.Unlock()
}

func (s *Status) OnChainDisconnect(f func()) {
	s.blockchain.onChainDisconnect = f
}

// Stop the internal checker(s) from periodically calling their underlying providers
// Note: currently the only way to start checking externally is to New up a new Status checker
func (s *Status) Stop() {
	s.blockchain.Stop()
}

func (s *Status) ChainStatus() types.ChainStatus {
	return s.blockchain.Status()
}

// Status returns the current status of the underlying Blockchain connection.
// Returned states are currently CONNECTED, REPLAYING or DISCONNECTED.
func (cs *ChainStatus) Status() types.ChainStatus {
	return types.ChainStatus(atomic.LoadUint32(&cs.status))
}

func (cs *ChainStatus) setStatus(status types.ChainStatus) {
	atomic.StoreUint32(&cs.status, uint32(status))
}

func (cs *ChainStatus) tick(status types.ChainStatus) types.ChainStatus {
	cs.cfgMu.Lock()
	defer cs.cfgMu.Unlock()
	newStatus := status
	_, err := cs.client.Health()
	if (status == types.ChainStatus_DISCONNECTED || status == types.ChainStatus_REPLAYING) && err == nil {
		cs.starting = false
		cs.retriesCount = int(cs.config.Retries)
		// node is connected, now let's check if we are replaying
		res, err2 := cs.client.GetStatus(context.Background())
		if err2 != nil {
			// error while getting the status, maybe we are not
			// really connected, let's not change the status then
			return status
		}

		if err := defaultChainVersion.Check(res.NodeInfo.Version); err != nil {
			cs.log.Error("tendermint version error",
				logging.Error(err),
			)
			cs.onChainDisconnect()
		}
		cs.log.Info("tendermint info",
			logging.String("version", res.NodeInfo.Version),
		)

		if res.SyncInfo.CatchingUp {
			newStatus = types.ChainStatus_REPLAYING
		} else {
			newStatus = types.ChainStatus_CONNECTED
		}
	} else if status == types.ChainStatus_CONNECTED && err != nil {
		newStatus = types.ChainStatus_DISCONNECTED
	}

	if status == types.ChainStatus_DISCONNECTED {
		cs.retriesCount -= 1
	}

	if status == newStatus {
		return status
	}

	cs.setStatus(newStatus)
	cs.log.Info("Blockchain status updated",
		logging.String("status-old", status.String()),
		logging.String("status-new", newStatus.String()))

	return newStatus
}

// Calling start ideally by go-routine will start periodic checking at interval specified by config
// until context cancel is triggered, typically by the stop function call.
func (cs *ChainStatus) start(ctx context.Context) {
	ticker := time.NewTicker(cs.config.Interval.Get())
	currentStatus := cs.Status()
	for {
		select {
		case <-ticker.C:
			currentStatus = cs.tick(currentStatus)
			// if status changed to disconnect, we try to call the onChainDisconnect
			// callback
			if currentStatus == types.ChainStatus_DISCONNECTED && cs.onChainDisconnect != nil && !cs.starting {
				cs.log.Info("Chain is disconnected, we'll try to reconnect",
					logging.Int("retries-left", cs.retriesCount),
				)

				if cs.retriesCount <= 0 {
					cs.cfgMu.Lock()
					cs.log.Info("Chain is still disconnected, shutting down now",
						logging.Int("retries-count", int(cs.config.Retries)),
					)
					cs.cfgMu.Unlock()
					cs.onChainDisconnect()
				}
			}

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
