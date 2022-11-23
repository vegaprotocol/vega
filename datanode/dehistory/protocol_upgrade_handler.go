package dehistory

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/logging"
)

type ProtocolUpgradeHandler struct {
	log                     *logging.Logger
	protocolUpgradeService  *service.ProtocolUpgrade
	createAndPublishSegment func(ctx context.Context, chainID string, toHeight int64) error
}

func NewProtocolUpgradeHandler(log *logging.Logger, protocolUpgradeService *service.ProtocolUpgrade,
	createAndPublishSegment func(ctx context.Context, chainID string, toHeight int64) error,
) *ProtocolUpgradeHandler {
	return &ProtocolUpgradeHandler{
		log:                     log.Named("protocol-upgrade-handler"),
		protocolUpgradeService:  protocolUpgradeService,
		createAndPublishSegment: createAndPublishSegment,
	}
}

func (t *ProtocolUpgradeHandler) OnProtocolUpgradeEvent(ctx context.Context, chainID string,
	lastCommittedBlockHeight int64,
) error {
	if err := t.createAndPublishSegment(ctx, chainID, lastCommittedBlockHeight); err != nil {
		return fmt.Errorf("failed to create and publish segment: %w", err)
	}

	t.protocolUpgradeService.SetProtocolUpgradeStarted()

	return nil
}

func (t *ProtocolUpgradeHandler) GetProtocolUpgradeStarted() bool {
	return t.protocolUpgradeService.GetProtocolUpgradeStarted()
}
