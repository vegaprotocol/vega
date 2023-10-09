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

package v1

import (
	"context"
	"time"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	v1 "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
)

type ConsentConfirmation struct {
	TxID     string
	Decision bool
}

type ConsentRequest struct {
	TxID         string
	Tx           *v1.SubmitTransactionRequest
	ReceivedAt   time.Time
	Confirmation chan ConsentConfirmation
}

type SentTransaction struct {
	TxHash string
	TxID   string
	Tx     *commandspb.Transaction
	Error  error
	SentAt time.Time
}

type Policy interface {
	Ask(tx *v1.SubmitTransactionRequest, txID string, receivedAt time.Time) (bool, error)
	Report(tx SentTransaction)
}

type AutomaticConsentPolicy struct{}

func NewAutomaticConsentPolicy() Policy {
	return &AutomaticConsentPolicy{}
}

func (p *AutomaticConsentPolicy) Ask(_ *v1.SubmitTransactionRequest, _ string, _ time.Time) (bool, error) {
	return true, nil
}

func (p *AutomaticConsentPolicy) Report(_ SentTransaction) {
	// Nothing to report as we expect this policy to be non-interactive.
}

type ExplicitConsentPolicy struct {
	// ctx is used to interrupt the wait for consent confirmation
	ctx context.Context

	consentRequestsChan  chan ConsentRequest
	sentTransactionsChan chan SentTransaction
}

func NewExplicitConsentPolicy(ctx context.Context, consentRequests chan ConsentRequest, sentTransactions chan SentTransaction) Policy {
	return &ExplicitConsentPolicy{
		ctx:                  ctx,
		consentRequestsChan:  consentRequests,
		sentTransactionsChan: sentTransactions,
	}
}

func (p *ExplicitConsentPolicy) Ask(tx *v1.SubmitTransactionRequest, txID string, receivedAt time.Time) (bool, error) {
	confirmationChan := make(chan ConsentConfirmation, 1)
	defer close(confirmationChan)

	consentRequest := ConsentRequest{
		TxID:         txID,
		Tx:           tx,
		ReceivedAt:   receivedAt,
		Confirmation: confirmationChan,
	}

	if err := p.sendConsentRequest(consentRequest); err != nil {
		return false, err
	}

	return p.receiveConsentConfirmation(consentRequest)
}

func (p *ExplicitConsentPolicy) receiveConsentConfirmation(consentRequest ConsentRequest) (bool, error) {
	for {
		select {
		case <-p.ctx.Done():
			return false, ErrInterruptedConsentRequest
		case decision := <-consentRequest.Confirmation:
			return decision.Decision, nil
		}
	}
}

func (p *ExplicitConsentPolicy) sendConsentRequest(consentRequest ConsentRequest) error {
	for {
		select {
		case <-p.ctx.Done():
			return ErrInterruptedConsentRequest
		case p.consentRequestsChan <- consentRequest:
			return nil
		}
	}
}

func (p *ExplicitConsentPolicy) Report(tx SentTransaction) {
	p.sentTransactionsChan <- tx
}
