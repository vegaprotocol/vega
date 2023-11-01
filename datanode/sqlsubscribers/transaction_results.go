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

// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/libs/slice"
	"code.vegaprotocol.io/vega/logging"
)

type TransactionResultEvent interface {
	events.Event
	TransactionResult() events.TransactionResult
}

type TransactionResults struct {
	subscriber
	observer utils.Observer[events.TransactionResult]
}

func NewTransactionResults(log *logging.Logger) *TransactionResults {
	return &TransactionResults{
		observer: utils.NewObserver[events.TransactionResult]("transaction_result", log, 5, 5),
	}
}

func (tr *TransactionResults) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case TransactionResultEvent:
		tr.observer.Notify([]events.TransactionResult{e.TransactionResult()})
		return nil
	default:
		return nil
	}
}
func (tr *TransactionResults) Types() []events.Type {
	return []events.Type{events.TransactionResultEvent}
}

func (tr *TransactionResults) Observe(ctx context.Context, retries int,
	partyIDs []string, hashes []string, status *bool,
) (transactions <-chan []events.TransactionResult, ref uint64) {
	ch, ref := tr.observer.Observe(ctx,
		retries,
		func(tre events.TransactionResult) bool {
			partiesOk := len(partyIDs) == 0 || slice.Contains(partyIDs, tre.PartyID())
			hashesOk := len(hashes) == 0 || slice.Contains(hashes, tre.Hash())
			statusOK := status == nil || *status == tre.Status()

			return partiesOk && hashesOk && statusOK
		})
	return ch, ref
}
