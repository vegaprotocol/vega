package banking

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
)

var ErrUnsupportedTransferKind = errors.New("unsupported transfer kind")

type scheduledTransfer struct {
	// to send events
	oneoff      *types.OneOffTransfer
	transfer    *types.Transfer
	accountType types.AccountType
	reference   string
}

func (s *scheduledTransfer) ToProto() *checkpoint.ScheduledTransfer {
	return &checkpoint.ScheduledTransfer{
		OneoffTransfer: s.oneoff.IntoEvent(),
		Transfer:       s.transfer.IntoProto(),
		AccountType:    s.accountType,
		Reference:      s.reference,
	}
}

func scheduledTransferFromProto(p *checkpoint.ScheduledTransfer) (scheduledTransfer, error) {
	transfer, err := types.TransferFromProto(p.Transfer)
	if err != nil {
		return scheduledTransfer{}, err
	}

	return scheduledTransfer{
		oneoff:      types.OneOffTransferFromEvent(p.OneoffTransfer),
		transfer:    transfer,
		accountType: p.AccountType,
		reference:   p.Reference,
	}, nil
}

func (e *Engine) oneOffTransfer(
	ctx context.Context,
	transfer *types.OneOffTransfer,
) error {
	defer func() {
		e.broker.Send(events.NewOneOffTransferFundsEvent(ctx, transfer))
	}()

	// ensure asset exists
	if _, err := e.assets.Get(transfer.Asset); err != nil {
		transfer.Status = types.TransferStatusRejected
		e.log.Debug("cannot transfer funds, invalid asset", logging.Error(err))
		return fmt.Errorf("could not transfer funds: %w", err)
	}

	if err := transfer.IsValid(); err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	tresps, err := e.processTransfer(
		ctx, transfer.From, transfer.To, transfer.Asset, transfer.FromAccountType,
		transfer.ToAccountType, transfer.Amount, transfer.Reference, transfer,
	)
	if err != nil {
		transfer.Status = types.TransferStatusRejected
		return err
	}

	// all was OK
	transfer.Status = types.TransferStatusDone
	e.broker.Send(events.NewTransferResponse(ctx, tresps))

	return nil
}

type timesToTransfers struct {
	deliverOn time.Time
	transfer  []scheduledTransfer
}

func (e *Engine) distributeScheduledTransfers(ctx context.Context) error {
	ttfs := []timesToTransfers{}

	// iterate over those scheduled transfers to sort them by time
	for k, v := range e.scheduledTransfers {
		if !e.currentTime.Before(k) {
			ttfs = append(ttfs, timesToTransfers{k, v})
			delete(e.scheduledTransfers, k)
		}
	}

	// sort slice by time.
	// no need to sort transfers they are going out as first in first out.
	sort.SliceStable(ttfs, func(i, j int) bool {
		return ttfs[i].deliverOn.Before(ttfs[j].deliverOn)
	})

	transfers := []*types.Transfer{}
	accountTypes := []types.AccountType{}
	references := []string{}
	evts := []events.Event{}
	for _, v := range ttfs {
		for _, t := range v.transfer {
			t.oneoff.Status = types.TransferStatusDone
			evts = append(evts, events.NewOneOffTransferFundsEvent(ctx, t.oneoff))
			transfers = append(transfers, t.transfer)
			accountTypes = append(accountTypes, t.accountType)
			references = append(references, t.reference)
		}
	}

	if len(transfers) <= 0 {
		// nothing to do yeay
		return nil
	}

	tresps, err := e.col.TransferFunds(
		ctx, transfers, accountTypes, references, nil, nil, // no fees required there, they've been paid already
	)
	if err != nil {
		return err
	}

	e.broker.Send(events.NewTransferResponse(ctx, tresps))
	e.broker.SendBatch(evts)

	return nil
}

func (e *Engine) scheduleTransfer(
	oneoff *types.OneOffTransfer,
	t *types.Transfer,
	ty types.AccountType,
	reference string,
	deliverOn time.Time,
) {
	sts, ok := e.scheduledTransfers[deliverOn]
	if !ok {
		e.scheduledTransfers[deliverOn] = []scheduledTransfer{}
		sts = e.scheduledTransfers[deliverOn]
	}

	sts = append(sts, scheduledTransfer{
		oneoff:      oneoff,
		transfer:    t,
		accountType: ty,
		reference:   reference,
	})
	e.scheduledTransfers[deliverOn] = sts
}
