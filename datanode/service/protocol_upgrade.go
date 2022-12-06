package service

import (
	"context"

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
}
type ProtocolUpgrade struct {
	pupStore       pupStore
	upgradeStarted bool
	log            *logging.Logger
}

func NewProtocolUpgrade(pupStore pupStore, log *logging.Logger) *ProtocolUpgrade {
	return &ProtocolUpgrade{
		pupStore: pupStore,
		log:      log,
	}
}

func (p *ProtocolUpgrade) GetProtocolUpgradeStarted() bool {
	return p.upgradeStarted
}

func (p *ProtocolUpgrade) SetProtocolUpgradeStarted() {
	p.log.Info("datanode is ready for protocol upgrade")
	p.upgradeStarted = true
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
