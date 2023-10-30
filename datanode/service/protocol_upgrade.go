// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"context"
	"sync/atomic"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
)

type pupStore interface {
	Add(ctx context.Context, p entities.ProtocolUpgradeProposal) error
	List(ctx context.Context,
		status *entities.ProtocolUpgradeProposalStatus,
		approvedBy *string,
		pagination entities.CursorPagination,
	) ([]entities.ProtocolUpgradeProposal, entities.PageInfo, error)
	GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.ProtocolUpgradeProposal, error)
}
type ProtocolUpgrade struct {
	pupStore pupStore

	// Flag update needs to be visible across threads
	upgradeStarted atomic.Bool
	log            *logging.Logger
}

func NewProtocolUpgrade(pupStore pupStore, log *logging.Logger) *ProtocolUpgrade {
	return &ProtocolUpgrade{
		pupStore: pupStore,
		log:      log,
	}
}

func (p *ProtocolUpgrade) GetProtocolUpgradeStarted() bool {
	return p.upgradeStarted.Load()
}

func (p *ProtocolUpgrade) SetProtocolUpgradeStarted() {
	p.log.Info("datanode is ready for protocol upgrade")
	p.upgradeStarted.Store(true)
}

func (p *ProtocolUpgrade) AddProposal(ctx context.Context, pup entities.ProtocolUpgradeProposal) error {
	return p.pupStore.Add(ctx, pup)
}

func (p *ProtocolUpgrade) ListProposals(ctx context.Context,
	status *entities.ProtocolUpgradeProposalStatus,
	approvedBy *string,
	pagination entities.CursorPagination,
) ([]entities.ProtocolUpgradeProposal, entities.PageInfo, error) {
	return p.pupStore.List(ctx, status, approvedBy, pagination)
}

func (p *ProtocolUpgrade) GetByTxHash(ctx context.Context, txHash entities.TxHash) ([]entities.ProtocolUpgradeProposal, error) {
	return p.pupStore.GetByTxHash(ctx, txHash)
}
