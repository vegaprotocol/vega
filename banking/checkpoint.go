package banking

import (
	"context"
	"sort"
	"time"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
)

func (e *Engine) Name() types.CheckpointName {
	return types.BankingCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	msg := &checkpoint.Banking{
		TransfersAtTime: e.getScheduledTransfers(),
	}
	ret, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	b := checkpoint.Banking{}
	if err := proto.Unmarshal(data, &b); err != nil {
		return err
	}

	for _, v := range b.TransfersAtTime {
		transfers := make([]scheduledTransfer, 0, len(v.Transfers))
		for _, v := range v.Transfers {
			transfer, err := scheduledTransferFromProto(v)
			if err != nil {
				return err
			}
			transfers = append(transfers, transfer)
		}
		e.scheduledTransfers[time.Unix(v.DeliverOn, 0)] = transfers
	}
	return nil
}

func (e *Engine) getScheduledTransfers() []*checkpoint.ScheduledTransferAtTime {
	out := []*checkpoint.ScheduledTransferAtTime{}

	for k, v := range e.scheduledTransfers {
		transfers := make([]*checkpoint.ScheduledTransfer, 0, len(v))
		for _, v := range v {
			transfers = append(transfers, v.ToProto())
		}

		out = append(out, &checkpoint.ScheduledTransferAtTime{DeliverOn: k.Unix(), Transfers: transfers})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].DeliverOn < out[j].DeliverOn })

	return out
}
