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

func (r *Delegation) GetDelegationSubscribersCount() int32 {
	return r.observer.GetSubscribersCount()
}
