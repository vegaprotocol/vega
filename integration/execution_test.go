package core_test

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/integration/stubs"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
)

// embeds the execution engine. Just forwards the calls and creates the TxErr events
// if any of the ingress methods returns an error (as the processor would).
type exEng struct {
	*execution.Engine
	broker *stubs.BrokerStub
}

func newExEng(e *execution.Engine, broker *stubs.BrokerStub) *exEng {
	return &exEng{
		Engine: e,
		broker: broker,
	}
}

func (e *exEng) SubmitOrder(ctx context.Context, submission *types.OrderSubmission, party string) (*types.OrderConfirmation, error) {
	conf, err := e.Engine.SubmitOrder(ctx, submission, party, vgcrypto.RandomHash())
	if err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, submission.IntoProto()))
	}
	return conf, err
}

func (e *exEng) AmendOrder(ctx context.Context, amendment *types.OrderAmendment, party string) (*types.OrderConfirmation, error) {
	conf, err := e.Engine.AmendOrder(ctx, amendment, party, vgcrypto.RandomHash())
	if err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, amendment.IntoProto()))
	}
	return conf, err
}

func (e *exEng) CancelOrder(ctx context.Context, cancel *types.OrderCancellation, party string) ([]*types.OrderCancellationConfirmation, error) {
	conf, err := e.Engine.CancelOrder(ctx, cancel, party, vgcrypto.RandomHash())
	if err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, cancel.IntoProto()))
	}
	return conf, err
}

func (e *exEng) SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, lpID,
	deterministicId string) error {
	if err := e.Engine.SubmitLiquidityProvision(ctx, sub, party, deterministicId); err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, sub.IntoProto()))
		return err
	}
	return nil
}

func (e *exEng) AmendLiquidityProvision(ctx context.Context, lpa *types.LiquidityProvisionAmendment, party string) error {
	if err := e.Engine.AmendLiquidityProvision(ctx, lpa, party, vgcrypto.RandomHash()); err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, lpa.IntoProto()))
		return err
	}
	return nil
}

func (e *exEng) CancelLiquidityProvision(ctx context.Context, lpc *types.LiquidityProvisionCancellation, party string) error {
	if err := e.Engine.CancelLiquidityProvision(ctx, lpc, party); err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, lpc.IntoProto()))
		return err
	}
	return nil
}
