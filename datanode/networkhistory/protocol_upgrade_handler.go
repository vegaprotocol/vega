package networkhistory

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/logging"
)

type eventSender interface {
	Send(events.Event) error
}

type ProtocolUpgradeHandler struct {
	log                     *logging.Logger
	protocolUpgradeService  *service.ProtocolUpgrade
	eventSender             eventSender
	createAndPublishSegment func(ctx context.Context, chainID string, toHeight int64) error
}

func NewProtocolUpgradeHandler(
	log *logging.Logger,
	protocolUpgradeService *service.ProtocolUpgrade,
	eventSender eventSender,
	createAndPublishSegment func(ctx context.Context, chainID string, toHeight int64) error,
) *ProtocolUpgradeHandler {
	return &ProtocolUpgradeHandler{
		log:                     log.Named("protocol-upgrade-handler"),
		protocolUpgradeService:  protocolUpgradeService,
		createAndPublishSegment: createAndPublishSegment,
		eventSender:             eventSender,
	}
}

func (t *ProtocolUpgradeHandler) OnProtocolUpgradeEvent(ctx context.Context, chainID string,
	lastCommittedBlockHeight int64,
) error {
	if err := t.createAndPublishSegment(ctx, chainID, lastCommittedBlockHeight); err != nil {
		t.log.Error("Failed to create and publish segment", logging.Error(err))
		return fmt.Errorf("failed to create and publish segment: %w", err)
	}

	t.log.Debug("Created and published segment", logging.Int64("last_committed_block_height", lastCommittedBlockHeight))

	t.protocolUpgradeService.SetProtocolUpgradeStarted()

	if err := t.eventSender.Send(events.NewProtocolUpgradeDataNodeReady(ctx, lastCommittedBlockHeight)); err != nil {
		t.log.Error("Failed to send data node upgrade event", logging.Error(err))
		return err
	}

	t.log.Debug("Notified Core about being ready for protocol upgrade", logging.Int64("last_committed_block_height", lastCommittedBlockHeight))

	return nil
}

func (t *ProtocolUpgradeHandler) GetProtocolUpgradeStarted() bool {
	return t.protocolUpgradeService.GetProtocolUpgradeStarted()
}
