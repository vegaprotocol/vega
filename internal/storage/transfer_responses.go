package storage

import (
	"fmt"
	"sync"

	"code.vegaprotocol.io/vega/internal/logging"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"
)

type TransferResponse struct {
	Config
	log          *logging.Logger
	subscribers  map[uint64]chan []*types.TransferResponse
	subscriberID uint64
	mu           sync.Mutex
}

func NewTransferResponses(log *logging.Logger, cfg Config) (*TransferResponse, error) {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &TransferResponse{
		Config:      cfg,
		log:         log,
		subscribers: map[uint64]chan []*types.TransferResponse{},
	}, nil
}

func (a *TransferResponse) ReloadConf(cfg Config) {
	a.log.Info("reloading configuration")
	if a.log.GetLevel() != cfg.Level.Get() {
		a.log.Info("updating log level",
			logging.String("old", a.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		a.log.SetLevel(cfg.Level.Get())
	}

	a.Config = cfg
}

func (a *TransferResponse) Close() error {
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
			break
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
	return
}

func (t *TransferResponse) Subscribe(c chan []*types.TransferResponse) uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.subscriberID += 1
	t.subscribers[t.subscriberID] = c

	t.log.Debug("TransferResponse subscriber added in transfer response store",
		logging.Uint64("subscriber-id", t.subscriberID))

	return t.subscriberID
}

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

	return errors.New(fmt.Sprintf("TransferResponse store subscriber does not exist with id: %d", id))
}

func (t *TransferResponse) SaveBatch(trs []*types.TransferResponse) error {
	t.notify(trs)
	return nil
}
