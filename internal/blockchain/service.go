package blockchain

import (
	"vega/msg"
	"time"
	"vega/vegatime"
	"vega/internal/execution"
	"github.com/pkg/errors"
	"fmt"
)

type Service interface {
	Begin() error
	Commit() error

	SubmitOrder(order *msg.Order) error
	CancelOrder(order *msg.Order) error
	AmendOrder(order *msg.Amendment) error
	ValidateOrder(order *msg.Order) error
}

type abciService struct {
	*Config
	*Stats

	execution         execution.Engine
	previousTimestamp vegatime.Stamp
	currentTimestamp  vegatime.Stamp
	previousDatetime  time.Time
	currentDatetime   time.Time

	ordersInBatchLengths []int
	currentOrdersInBatch int
	currentTradesInBatch int
	totalBatches uint64
	totalOrders  uint64
	totalTrades  uint64
}

func NewAbciService(conf *Config, stats *Stats, ex execution.Engine) Service {
	return &abciService{Config: conf, Stats: stats, execution: ex}
}

func (s *abciService) Begin() error {
	s.log.Debug("AbciService: Begin")

	return nil
}

func (s *abciService) ValidateOrder(order *msg.Order) error {
	s.log.Debug("AbciService: Validating order")

	return nil
}

func (s *abciService) Commit() error {
	s.log.Debug("AbciService: Commit")

	s.setBatchStats()
	return nil
}

func (s *abciService) SubmitOrder(order *msg.Order) error {
	if s.logOrderSubmitDebug {
		s.log.Debugf("AbciService: received a SUBMIT ORDER request: %s", order)
	}

	order.Id = fmt.Sprintf("V%010d-%010d", s.totalBatches, s.totalOrders)
	order.Timestamp = s.currentTimestamp.UnixNano()
	
	// Submit the create order request to the execution engine
	confirmationMessage, errorMessage := s.execution.SubmitOrder(order)
	if confirmationMessage != nil {
		if s.logOrderSubmitDebug {
			s.log.Debugf("Order confirmation message:")
			s.log.Debugf("- aggressive order: %+v", confirmationMessage.Order)
			s.log.Debugf("- trades: %+v", confirmationMessage.Trades)
			s.log.Debugf("- passive orders affected: %+v", confirmationMessage.PassiveOrdersAffected)
			s.totalTrades += uint64(len(confirmationMessage.Trades))
		}

		confirmationMessage.Release()
	}

	// increment total orders, even for failures so current ID strategy is valid.
	s.totalOrders++

	if errorMessage != msg.OrderError_NONE {
		s.log.Errorf("ABCI order error message (create):")
		s.log.Errorf("- error: %s", errorMessage)
		return errors.New(errorMessage.String())
	}

	return nil
}

func (s *abciService) CancelOrder(order *msg.Order) error {
	if s.logOrderCancelDebug {
		s.log.Debugf("AbciService: received a CANCEL ORDER request")
	}

	// Submit the cancel new order request to the Vega trading core
	cancellationMessage, errorMessage := s.execution.CancelOrder(order)
	if cancellationMessage != nil {
		if s.logOrderCancelDebug {
			s.log.Debugf("ABCI order cancellation message:")
			s.log.Debugf("- cancelled order: %+v", cancellationMessage.Order)
		}
	}
	if errorMessage != msg.OrderError_NONE {
		s.log.Errorf("ABCI order error message (cancel):")
		s.log.Errorf("- error: %s", errorMessage.String())
		return errors.New(errorMessage.String())
	}

	return nil
}

func (s *abciService) AmendOrder(order *msg.Amendment) error {
	if s.logOrderAmendDebug {
		s.log.Debugf("AbciService: received a AMEND ORDER request")
	}
	
	// Submit the Amendment new order request to the Vega trading core
	confirmationMessage, errorMessage := s.execution.AmendOrder(order)
	if confirmationMessage != nil {
		if s.logOrderAmendDebug {
			s.log.Debugf("AbciService: Amend order from execution engine:")
			s.log.Debugf("- cancelled order: %+v\n", confirmationMessage.Order)
		}
	}
	if errorMessage != msg.OrderError_NONE {
		s.log.Errorf("AbciService: Amend order error from execution engine:")
		s.log.Errorf("- error: %s", errorMessage.String())
		return errors.New(errorMessage.String())
	}
	
	return nil
}

func (s *abciService) setBatchStats() {
	s.totalBatches++
	s.totalOrdersLastBatch = s.currentOrdersInBatch
	s.totalTradesLastBatch = s.currentTradesInBatch

	// Calculate total orders per batch (per block) average
	if s.currentOrdersInBatch > 0 {
		if s.ordersInBatchLengths == nil {
			s.ordersInBatchLengths = make([]int, 0)
		}
		s.ordersInBatchLengths = append(s.ordersInBatchLengths, s.currentOrdersInBatch)
		lenOrdersInBatch := len(s.ordersInBatchLengths)
		if lenOrdersInBatch > 0 {
			totalOrders := 0
			for _, itx := range s.ordersInBatchLengths {
				totalOrders += itx
			}
			s.averageOrdersPerBatch = totalOrders / lenOrdersInBatch

			// MAX sample size for avg calculation is 5000
			if lenOrdersInBatch == 5000 {
				s.ordersInBatchLengths = nil
			}
		}
	}

	s.log.Debugf("AbciService: Average orders/batch = %v", s.averageOrdersPerBatch)

	s.currentOrdersInBatch = 0
	s.currentTradesInBatch = 0
}

