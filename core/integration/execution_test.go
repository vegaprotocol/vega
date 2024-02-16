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

package core_test

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/processor"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/ptr"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

var (
	ErrBatchInstructionNotStarted                = errors.New("batch instruction not started")
	ErrBatchInstructionStartedWithDifferentParty = errors.New("batch instruction started with different party")
)

type batchIntruction struct {
	party string
	bmi   *commandspb.BatchMarketInstructions
}

// embeds the execution engine. Just forwards the calls and creates the TxErr events
// if any of the ingress methods returns an error (as the processor would).
type exEng struct {
	*execution.Engine
	broker *stubs.BrokerStub
	batch  *batchIntruction
}

func newExEng(e *execution.Engine, broker *stubs.BrokerStub) *exEng {
	return &exEng{
		Engine: e,
		broker: broker,
	}
}

func (e *exEng) BlockEnd(ctx context.Context) {
	// set hash ID to some value
	ctx = vgcontext.WithTraceID(ctx, "deadbeef")
	e.Engine.BlockEnd(ctx)
}

func (e *exEng) SubmitOrder(ctx context.Context, submission *types.OrderSubmission, party string) (*types.OrderConfirmation, error) {
	idgen := idgeneration.New(vgcrypto.RandomHash())
	conf, err := e.Engine.SubmitOrder(ctx, submission, party, idgen, idgen.NextID())
	if err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, submission.IntoProto(), "submitOrder"))
	}
	return conf, err
}

func (e *exEng) SubmitStopOrder(
	ctx context.Context,
	submission *types.StopOrdersSubmission,
	party string,
) (*types.OrderConfirmation, error) {
	idgen := idgeneration.New(vgcrypto.RandomHash())
	var fallsBelowID, risesAboveID *string
	if submission.FallsBelow != nil {
		fallsBelowID = ptr.From(idgen.NextID())
	}
	if submission.RisesAbove != nil {
		risesAboveID = ptr.From(idgen.NextID())
	}
	conf, err := e.Engine.SubmitStopOrders(ctx, submission, party, idgen, fallsBelowID, risesAboveID)
	// if err != nil {
	// 	e.broker.Send(events.NewTxErrEvent(ctx, err, party, submission.IntoProto(), "submitOrder"))
	// }
	return conf, err
}

func (e *exEng) AmendOrder(ctx context.Context, amendment *types.OrderAmendment, party string) (*types.OrderConfirmation, error) {
	idgen := idgeneration.New(vgcrypto.RandomHash())
	conf, err := e.Engine.AmendOrder(ctx, amendment, party, idgen)
	if err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, amendment.IntoProto(), "amendOrder"))
	}
	return conf, err
}

func (e *exEng) CancelOrder(ctx context.Context, cancel *types.OrderCancellation, party string) ([]*types.OrderCancellationConfirmation, error) {
	idgen := idgeneration.New(vgcrypto.RandomHash())
	conf, err := e.Engine.CancelOrder(ctx, cancel, party, idgen)
	if err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, cancel.IntoProto(), "cancelOrder"))
	}
	return conf, err
}

func (e *exEng) CancelStopOrder(ctx context.Context, cancel *types.StopOrdersCancellation, party string) error {
	idgen := idgeneration.New(vgcrypto.RandomHash())
	return e.Engine.CancelStopOrders(ctx, cancel, party, idgen)
}

func (e *exEng) SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, lpID,
	deterministicID string,
) error {
	if err := e.Engine.SubmitLiquidityProvision(ctx, sub, party, deterministicID); err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, sub.IntoProto(), "submitLiquidityProvision"))
		return err
	}
	return nil
}

func (e *exEng) AmendLiquidityProvision(ctx context.Context, lpa *types.LiquidityProvisionAmendment, party string) error {
	if err := e.Engine.AmendLiquidityProvision(ctx, lpa, party, vgcrypto.RandomHash()); err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, lpa.IntoProto(), "amendLiquidityProvision"))
		return err
	}
	return nil
}

func (e *exEng) CancelLiquidityProvision(ctx context.Context, lpc *types.LiquidityProvisionCancellation, party string) error {
	if err := e.Engine.CancelLiquidityProvision(ctx, lpc, party); err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, lpc.IntoProto(), "cancelLiquidityProvision"))
		return err
	}
	return nil
}

// batch order bits that sit above the exeuction engine.
func (e *exEng) StartBatch(party string) error {
	e.batch = &batchIntruction{
		party: party,
		bmi:   &commandspb.BatchMarketInstructions{},
	}
	return nil
}

func (e *exEng) AddSubmitOrderToBatch(order *types.OrderSubmission, party string) error {
	if e.batch == nil {
		return ErrBatchInstructionNotStarted
	}
	if e.batch.party != party {
		return ErrBatchInstructionStartedWithDifferentParty
	}
	e.batch.bmi.Submissions = append(e.batch.bmi.Submissions, order.IntoProto())
	return nil
}

func (e *exEng) ProcessBatch(ctx context.Context, party string) error {
	if e.batch == nil {
		return ErrBatchInstructionNotStarted
	}
	if e.batch.party != party {
		return ErrBatchInstructionStartedWithDifferentParty
	}

	batch := e.batch.bmi
	e.batch = nil
	bmi := processor.NewBMIProcessor(nil, e.Engine, noopValidation{})
	if err := bmi.ProcessBatch(context.Background(), batch, party, vgcrypto.RandomHash(), stats.NewBlockchain()); err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, nil, "processBatch"))
		return err
	}
	return nil
}

func (e *exEng) SubmitAMM(ctx context.Context, submission *types.SubmitAMM) error {
	idgen := idgeneration.New(vgcrypto.RandomHash())
	if err := e.Engine.SubmitAMM(ctx, submission, idgen.NextID()); err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, submission.Party, submission.IntoProto(), "submitAMM"))
		return err
	}
	return nil
}

func (e *exEng) AmendAMM(ctx context.Context, submission *types.AmendAMM) error {
	if err := e.Engine.AmendAMM(ctx, submission); err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, submission.Party, submission.IntoProto(), "amendAMM"))
		return err
	}
	return nil
}

func (e *exEng) CancelAMM(ctx context.Context, cancel *types.CancelAMM) error {
	idgen := idgeneration.New(vgcrypto.RandomHash())
	if err := e.Engine.CancelAMM(ctx, cancel, idgen.NextID()); err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, cancel.Party, cancel.IntoProto(), "cancelAMM"))
		return err
	}
	return nil
}

type noopValidation struct{}

func (n noopValidation) CheckOrderCancellation(cancel *commandspb.OrderCancellation) error {
	return nil
}

func (n noopValidation) CheckOrderAmendment(amend *commandspb.OrderAmendment) error {
	return nil
}

func (n noopValidation) CheckOrderSubmission(order *commandspb.OrderSubmission) error {
	return nil
}

func (n noopValidation) CheckStopOrdersCancellation(cancel *commandspb.StopOrdersCancellation) error {
	return nil
}

func (n noopValidation) CheckStopOrdersSubmission(order *commandspb.StopOrdersSubmission) error {
	return nil
}

func (n noopValidation) CheckUpdateMarginMode(order *commandspb.UpdateMarginMode) error {
	return nil
}
