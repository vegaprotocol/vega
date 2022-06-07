package service

import (
	"context"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/utils"
)

type delegationStore interface {
	Add(ctx context.Context, d entities.Delegation) error
	GetAll(ctx context.Context) ([]entities.Delegation, error)
	Get(ctx context.Context, partyID *string, nodeID *string, epoch *int64, p *entities.OffsetPagination) ([]entities.Delegation, error)
}

type Delegation struct {
	log      *logging.Logger
	store    delegationStore
	observer utils.Observer[entities.Delegation]
}

func NewDelegation(store delegationStore, log *logging.Logger) *Delegation {
	return &Delegation{
		store:    store,
		log:      log,
		observer: utils.NewObserver[entities.Delegation]("delegation", log, 10, 10),
	}
}

func (d *Delegation) Add(ctx context.Context, delegation entities.Delegation) error {
	err := d.store.Add(ctx, delegation)
	if err != nil {
		return err
	}
	d.observer.Notify([]entities.Delegation{delegation})
	return nil
}

func (d *Delegation) GetAll(ctx context.Context) ([]entities.Delegation, error) {
	return d.store.GetAll(ctx)
}

func (d *Delegation) Get(ctx context.Context, partyID *string, nodeID *string, epoch *int64, p *entities.OffsetPagination) ([]entities.Delegation, error) {
	return d.store.Get(ctx, partyID, nodeID, epoch, p)
}

func (r *Delegation) Observe(ctx context.Context, retries int, partyID, nodeID string) (rewardCh <-chan []entities.Delegation, ref uint64) {
	ch, ref := r.observer.Observe(ctx,
		retries,
		func(dele entities.Delegation) bool {
			return (len(nodeID) == 0 || nodeID == dele.NodeID.String()) &&
				(len(partyID) == 0 || partyID == dele.PartyID.String())
		})
	return ch, ref
}
