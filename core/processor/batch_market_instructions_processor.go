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

package processor

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type Validate struct{}

func (v Validate) CheckOrderCancellation(cancel *commandspb.OrderCancellation) error {
	return commands.CheckOrderCancellation(cancel)
}

func (v Validate) CheckOrderAmendment(amend *commandspb.OrderAmendment) error {
	return commands.CheckOrderAmendment(amend)
}

func (v Validate) CheckOrderSubmission(order *commandspb.OrderSubmission) error {
	return commands.CheckOrderSubmission(order)
}

func (v Validate) CheckStopOrdersCancellation(cancel *commandspb.StopOrdersCancellation) error {
	return commands.CheckStopOrdersCancellation(cancel)
}

func (v Validate) CheckStopOrdersSubmission(order *commandspb.StopOrdersSubmission) error {
	return commands.CheckStopOrdersSubmission(order)
}

func (v Validate) CheckUpdateMarginMode(order *commandspb.UpdateMarginMode) error {
	return commands.CheckUpdateMarginMode(order)
}

type Validator interface {
	CheckOrderCancellation(cancel *commandspb.OrderCancellation) error
	CheckOrderAmendment(amend *commandspb.OrderAmendment) error
	CheckOrderSubmission(order *commandspb.OrderSubmission) error
	CheckStopOrdersCancellation(cancel *commandspb.StopOrdersCancellation) error
	CheckStopOrdersSubmission(order *commandspb.StopOrdersSubmission) error
	CheckUpdateMarginMode(update *commandspb.UpdateMarginMode) error
}

type BMIProcessor struct {
	log       *logging.Logger
	exec      ExecutionEngine
	validator Validator
}

func NewBMIProcessor(
	log *logging.Logger,
	exec ExecutionEngine,
	validator Validator,
) *BMIProcessor {
	return &BMIProcessor{
		log:       log,
		exec:      exec,
		validator: validator,
	}
}

// BMIError implements blockchain/abci.MaybePartialError.
type BMIError struct {
	commands.Errors
	Partial bool
}

func (e *BMIError) GetRawErrors() map[string][]error {
	return e.Errors
}

func (e *BMIError) IsPartial() bool {
	return e.Partial
}

func (e *BMIError) Error() string {
	return e.Errors.Error()
}

func (e *BMIError) ErrorOrNil() error {
	if len(e.Errors) <= 0 {
		return nil
	}
	return e
}

// ProcessBatch will process a batch of market transaction. Transaction are
// always executed in the following order: cancellation, amendment then submissions.
// All errors are returned as a single error.
func (p *BMIProcessor) ProcessBatch(
	ctx context.Context,
	batch *commandspb.BatchMarketInstructions,
	party, determinitisticID string,
	stats Stats,
) error {
	errs := &BMIError{
		Errors: commands.NewErrors(),
	}
	// keep track of the index of the current instruction
	// in the whole batch e.g:
	// a batch with 10 instruction in each array
	// idx 11 will be the second instruction of the
	// amendment array
	idx := 0

	// keep track of the cnt of txn which errored to customise
	// returned error from ABCI
	errCnt := 0

	// first we generate the IDs for all new orders,
	// these need to be determinitistic
	idgen := idgeneration.New(determinitisticID)

	failedMarkets := map[string]error{}
	for _, umm := range batch.UpdateMarginMode {
		err := p.validator.CheckUpdateMarginMode(umm)
		if err == nil {
			var marginFactor num.Decimal
			if umm.MarginFactor == nil || len(*umm.MarginFactor) == 0 {
				marginFactor = num.DecimalZero()
			} else {
				marginFactor = num.MustDecimalFromString(*umm.MarginFactor)
			}
			err = p.exec.UpdateMarginMode(ctx, party, umm.MarketId, vega.MarginMode(umm.Mode), marginFactor)
		}
		if err != nil {
			errs.AddForProperty("updateMarginMode", err)
			errCnt++
			failedMarkets[umm.MarketId] = fmt.Errorf("Update margin mode transaction failed for market %s. Ignoring all transactions for the market", umm.MarketId)
		}
	}

	// each order will need a new ID, and each stop order can contain up to two orders (rises above, falls below)
	// but a stop order could also be invalid and have neither, so we pre-generate the maximum ids we might need
	nIDs := len(batch.Submissions) + (2 * len(batch.StopOrdersSubmission))
	submissionsIDs := make([]string, 0, nIDs)
	for i := 0; i < nIDs; i++ {
		submissionsIDs = append(submissionsIDs, idgen.NextID())
	}

	// process cancellations
	for i, cancel := range batch.Cancellations {
		err := p.validator.CheckOrderCancellation(cancel)
		if err == nil {
			if err, ok := failedMarkets[cancel.MarketId]; ok {
				errs.AddForProperty(fmt.Sprintf("%d", i), err)
				errCnt++
				idx++
				continue
			}
			stats.IncTotalCancelOrder()
			_, err = p.exec.CancelOrder(
				ctx, types.OrderCancellationFromProto(cancel), party, idgen)
		}

		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%d", i), err)
			errCnt++
		}
		idx++
	}

	// keep track of all amends already done, it's not legal to amend twice the
	// same order
	amended := map[string]struct{}{}

	// then amendments
	for _, protoAmend := range batch.Amendments {
		var err error
		if _, ok := amended[protoAmend.OrderId]; ok {
			// order already amended, just set an error, and do nothing
			err = errors.New("order already amended in batch")
		} else {
			err = p.validator.CheckOrderAmendment(protoAmend)
			if err == nil {
				if err, ok := failedMarkets[protoAmend.MarketId]; ok {
					errs.AddForProperty(fmt.Sprintf("%d", idx), err)
					errCnt++
					idx++
					continue
				}
				stats.IncTotalAmendOrder()
				var amend *types.OrderAmendment
				amend, err = types.NewOrderAmendmentFromProto(protoAmend)
				if err == nil {
					_, err = p.exec.AmendOrder(ctx, amend, party, idgen)
				}
			}
		}

		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%d", idx), err)
			errCnt++
		} else {
			// add to the amended list, a successful amend should prevent
			// any following amend of the same order
			amended[protoAmend.OrderId] = struct{}{}
		}
		idx++
	}

	// then submissions
	idIdx := 0
	for i, protoSubmit := range batch.Submissions {
		err := p.validator.CheckOrderSubmission(protoSubmit)
		if err == nil {
			var submit *types.OrderSubmission
			if err, ok := failedMarkets[protoSubmit.MarketId]; ok {
				errs.AddForProperty(fmt.Sprintf("%d", idx), err)
				errCnt++
				idx++
				continue
			}
			stats.IncTotalCreateOrder()
			if submit, err = types.NewOrderSubmissionFromProto(protoSubmit); err == nil {
				var conf *types.OrderConfirmation
				conf, err = p.exec.SubmitOrder(ctx, submit, party, idgen, submissionsIDs[i])
				if conf != nil {
					stats.AddCurrentTradesInBatch(uint64(len(conf.Trades)))
					stats.AddTotalTrades(uint64(len(conf.Trades)))
					stats.IncCurrentOrdersInBatch()
				}
				stats.IncTotalOrders()
			}
		}

		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%d", idx), err)
			errCnt++
		}
		idx++
		idIdx = i
	}

	// process cancellations
	for i, cancel := range batch.StopOrdersCancellation {
		err := p.validator.CheckStopOrdersCancellation(cancel)
		if err == nil {
			if err, ok := failedMarkets[*cancel.MarketId]; ok {
				errs.AddForProperty(fmt.Sprintf("%d", i), err)
				errCnt++
				idx++
				continue
			}
			stats.IncTotalCancelOrder()
			err = p.exec.CancelStopOrders(
				ctx, types.NewStopOrderCancellationFromProto(cancel), party, idgen)
		}

		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%d", i), err)
			errCnt++
		}
		idx++
	}

	for i, protoSubmit := range batch.StopOrdersSubmission {
		err := p.validator.CheckStopOrdersSubmission(protoSubmit)
		if err == nil {
			var submit *types.StopOrdersSubmission
			if protoSubmit.RisesAbove != nil && protoSubmit.RisesAbove.OrderSubmission != nil {
				if err, ok := failedMarkets[protoSubmit.RisesAbove.OrderSubmission.MarketId]; ok {
					errs.AddForProperty(fmt.Sprintf("%d", i), err)
					errCnt++
					idx++
					continue
				}
			}
			if protoSubmit.FallsBelow != nil && protoSubmit.FallsBelow.OrderSubmission != nil {
				if err, ok := failedMarkets[protoSubmit.FallsBelow.OrderSubmission.MarketId]; ok {
					errs.AddForProperty(fmt.Sprintf("%d", i), err)
					errCnt++
					idx++
					continue
				}
			}
			stats.IncTotalCreateOrder()
			if submit, err = types.NewStopOrderSubmissionFromProto(protoSubmit); err == nil {
				var id1, id2 *string
				var inc bool
				if submit.FallsBelow != nil {
					id1 = ptr.From(submissionsIDs[i+idIdx])
					inc = true
				}
				if submit.RisesAbove != nil {
					if inc {
						idIdx++
					}
					id2 = ptr.From(submissionsIDs[i+idIdx])
				}

				conf, err := p.exec.SubmitStopOrders(ctx, submit, party, idgen, id1, id2)
				if err == nil && conf != nil {
					stats.AddCurrentTradesInBatch(uint64(len(conf.Trades)))
					stats.AddTotalTrades(uint64(len(conf.Trades)))
					stats.IncCurrentOrdersInBatch()
					stats.IncTotalOrders()
				}
			}
		}

		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%d", idx), err)
			errCnt++
		}
		idx++
	}

	errs.Partial = errCnt != len(batch.UpdateMarginMode)+len(batch.Submissions)+len(batch.Amendments)+len(batch.Cancellations)+len(batch.StopOrdersCancellation)+len(batch.StopOrdersSubmission)

	return errs.ErrorOrNil()
}
