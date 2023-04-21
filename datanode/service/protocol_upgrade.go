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
