package appstatus

import (
	"context"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/proto"
)

type AppStatus struct {
	chainclt blockchain.Client
	log      *logging.Logger
	status   uint32

	cancel func()
}

func New(log *logging.Logger, clt blockchain.Client) *AppStatus {
	ctx, cancel := context.WithCancel(context.Background())
	as := &AppStatus{
		chainclt: clt,
		log:      log,
		status:   uint32(proto.AppStatus_DISCONNECTED),
		cancel:   cancel,
	}

	go as.updateStatus(ctx)
	return as
}

func (as *AppStatus) set(s proto.AppStatus) {
	atomic.StoreUint32(&as.status, uint32(s))
}

func (as *AppStatus) Get() proto.AppStatus {
	return proto.AppStatus(atomic.LoadUint32(&as.status))
}

func (as *AppStatus) updateStatus(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	currentStatus := as.Get()
	for {
		select {
		case <-ticker.C:
			oldStatus := currentStatus
			res, err := as.chainclt.Health()
			if currentStatus == proto.AppStatus_DISCONNECTED && err == nil {
				// node is connected, now let's check if we are replaying
				res, err := as.chainclt.GetStatus(context.Background())
				if err != nil {
					// error while getting the status, maybe we are not
					// really connected, let's not change the status then
					continue
				}
				if res.SyncInfo.CatchingUp {
					currentStatus = proto.AppStatus_CHAIN_REPLAYING
					as.set(proto.AppStatus_CONNECTED)
					continue
				}
				currentStatus = proto.AppStatus_CONNECTED
				as.set(proto.AppStatus_CONNECTED)
			} else if currentStatus == proto.AppStatus_CONNECTED && err != nil {
				currentStatus = proto.AppStatus_DISCONNECTED
				as.set(proto.AppStatus_DISCONNECTED)
			} else {
				// no status changes
				continue
			}
			as.log.Info("Application status updated",
				logging.String("status.old", oldStatus.String()),
				logging.String("status.new", currentStatus.String()))

		case <-ctx.Done():
			return
		}
	}
}

func (as *AppStatus) Stop() {
	as.cancel()
}
