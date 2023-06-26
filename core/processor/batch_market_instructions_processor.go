package processor

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
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

type Validator interface {
	CheckOrderCancellation(cancel *commandspb.OrderCancellation) error
	CheckOrderAmendment(amend *commandspb.OrderAmendment) error
	CheckOrderSubmission(order *commandspb.OrderSubmission) error
	CheckStopOrdersCancellation(cancel *commandspb.StopOrdersCancellation) error
	CheckStopOrdersSubmission(order *commandspb.StopOrdersSubmission) error
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
	isPartial bool
}

func (e *BMIError) IsPartial() bool {
	return e.isPartial
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

	// we will need an ID for every stop orders
	// not all StopsOrderSubmission have 2 StopOrder
	stopOrderSize := 0
	for _, v := range batch.StopOrdersSubmission {
		if v.FallsBelow != nil {
			stopOrderSize++
		}
		if v.RisesAbove != nil {
			stopOrderSize++
		}
	}

	submissionsIDs := make([]string, 0, len(batch.Submissions)+stopOrderSize)
	for i := 0; i < len(batch.Submissions); i++ {
		submissionsIDs = append(submissionsIDs, idgen.NextID())
	}

	// process cancellations
	for i, cancel := range batch.Cancellations {
		err := p.validator.CheckOrderCancellation(cancel)
		if err == nil {
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
				err = p.exec.SubmitStopOrders(ctx, submit, party, idgen, id1, id2)
				// FIXME(): when prototyp is updated, add the eventual crossing orders here.
				// if conf != nil {
				// 	stats.AddCurrentTradesInBatch(uint64(len(conf.Trades)))
				// 	stats.AddTotalTrades(uint64(len(conf.Trades)))
				// 	stats.IncCurrentOrdersInBatch()
				// }
				// stats.IncTotalOrders()
			}
		}

		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%d", idx), err)
			errCnt++
		}
		idx++
	}

	errs.isPartial = errCnt != len(batch.Submissions)+len(batch.Amendments)+len(batch.Cancellations)+len(batch.StopOrdersCancellation)+len(batch.StopOrdersSubmission)

	return errs.ErrorOrNil()
}
