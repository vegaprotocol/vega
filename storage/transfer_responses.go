package storage

import (
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

// TransferResponse is responsible for storing the ledger entries
type TransferResponse struct {
	Config
	log          *logging.Logger
	subscribers  map[uint64]chan []*types.TransferResponse
	subscriberID uint64
	mu           sync.Mutex
}

// NewTransferResponses instantiate a new TransferResponse
func NewTransferResponses(log *logging.Logger, cfg Config) (*TransferResponse, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &TransferResponse{
		Config:      cfg,
		log:         log,
		subscribers: map[uint64]chan []*types.TransferResponse{},
	}, nil
}

// ReloadConf update the internal configuration of the transfer responses
func (t *TransferResponse) ReloadConf(cfg Config) {
	t.log.Info("reloading configuration")
	if t.log.GetLevel() != cfg.Level.Get() {
		t.log.Info("updating log level",
			logging.String("old", t.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		t.log.SetLevel(cfg.Level.Get())
	}

	t.Config = cfg
}

// Close the underlying storage
func (t *TransferResponse) Close() error {
	// nothing to do at the moment, just keep it in par with the other store apis
	return nil
}

func (t *TransferResponse) notify(trs []*types.TransferResponse) {
	if len(trs) == 0 {
		return
	}

	t.mu.Lock()
	if len(t.subscribers) == 0 {
		t.log.Debug("No subscribers connected in TransferResponse store")
		t.mu.Unlock()
		return
	}

	var ok bool
	for id, sub := range t.subscribers {
		select {
		case sub <- trs:
			ok = true
		default:
			ok = false
		}
		if ok {
			t.log.Debug("TransferResponses channel updated for subscriber successfully",
				logging.Uint64("id", id))
		} else {
			t.log.Debug("TransferResponses channel could not be updated for subscriber",
				logging.Uint64("id", id))
		}
	}
	t.mu.Unlock()
}

// Subscribe add a new subscriber to the transfer response updates stream
func (t *TransferResponse) Subscribe(c chan []*types.TransferResponse) uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.subscriberID++
	t.subscribers[t.subscriberID] = c

	t.log.Debug("TransferResponse subscriber added in transfer response store",
		logging.Uint64("subscriber-id", t.subscriberID))

	return t.subscriberID
}

// Unsubscribe remove a subscriber from the transfer response updates stream
func (t *TransferResponse) Unsubscribe(id uint64) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.subscribers) == 0 {
		t.log.Debug("Un-subscribe called in transfer response store, no subscribers connected",
			logging.Uint64("subscriber-id", id))
		return nil
	}

	if _, exists := t.subscribers[id]; exists {
		delete(t.subscribers, id)

		t.log.Debug("Un-subscribe called in transfer response store, subscriber removed",
			logging.Uint64("subscriber-id", id))

		return nil
	}

	t.log.Warn("Un-subscribe called in transfer response store, subscriber does not exist",
		logging.Uint64("subscriber-id", id))

	return fmt.Errorf("subscriber to TransferResponse store does not exist with id: %d", id)
}

// SaveBatch save a new batch of transfer response
func (t *TransferResponse) SaveBatch(trs []*types.TransferResponse) error {
	t.notify(trs)
	return nil
}
