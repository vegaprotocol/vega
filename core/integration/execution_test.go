// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package core_test

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
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
	idgen := idgeneration.New(vgcrypto.RandomHash())
	conf, err := e.Engine.SubmitOrder(ctx, submission, party, idgen, idgen.NextID())
	if err != nil {
		e.broker.Send(events.NewTxErrEvent(ctx, err, party, submission.IntoProto(), "submitOrder"))
	}
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
