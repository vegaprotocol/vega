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

package sqlsubscribers_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers"
	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestTransactionResults(t *testing.T) {
	logger := logging.NewTestLogger()

	ctx := context.Background()

	subscriber := sqlsubscribers.NewTransactionResults(logger)

	expectedEvents := generateTestTransactionResultEvents(5, 5)

	var wg sync.WaitGroup
	// expect all events + success events + failed events + 2 per party events + 1 specific hash event
	wg.Add(len(expectedEvents)*2 + 3)

	// all events
	subChan, ref := subscriber.Observe(ctx, 2, []string{""}, []string{""}, nil)
	assert.Equal(t, uint64(1), ref)
	go func() {
		for events := range subChan {
			for range events {
				wg.Done()
			}
		}
	}()

	// success events
	success := true
	successSubChan, ref := subscriber.Observe(ctx, 2, []string{""}, []string{""}, &success)
	assert.Equal(t, uint64(2), ref)
	go func() {
		for events := range successSubChan {
			for range events {
				wg.Done()
			}
		}
	}()

	// failed events
	failure := false
	failureSubChan, ref := subscriber.Observe(ctx, 2, []string{""}, []string{""}, &failure)
	assert.Equal(t, uint64(3), ref)
	go func() {
		for events := range failureSubChan {
			for range events {
				wg.Done()
			}
		}
	}()

	// 2 per party events
	partySubChan, ref := subscriber.Observe(ctx, 2, []string{"party-2"}, []string{""}, nil)
	assert.Equal(t, uint64(4), ref)
	go func() {
		for events := range partySubChan {
			for range events {
				wg.Done()
			}
		}
	}()

	// 1 specific hash event
	hashSubChan, ref := subscriber.Observe(ctx, 2, []string{""}, []string{"hash-6"}, nil)
	assert.Equal(t, uint64(5), ref)
	go func() {
		for events := range hashSubChan {
			for range events {
				wg.Done()
			}
		}
	}()

	for _, event := range expectedEvents {
		subscriber.Push(context.Background(), event)
	}

	wg.Done()
}

func generateTestTransactionResultEvents(successCount, failureCount int) []*events.TransactionResult {
	out := make([]*events.TransactionResult, 0, successCount+failureCount)

	for i := 0; i < successCount; i++ {
		out = append(out,
			events.NewTransactionResultEventSuccess(
				context.Background(),
				fmt.Sprintf("hash-%d", i),
				fmt.Sprintf("party-%d", i),
				&commandspb.LiquidityProvisionSubmission{
					MarketId:         fmt.Sprintf("market-%d", i),
					CommitmentAmount: "100",
					Fee:              "1",
					Reference:        fmt.Sprintf("lp-%d", i),
				},
			),
		)
	}

	for i := 0; i < failureCount; i++ {
		nth := i + successCount
		out = append(out,
			events.NewTransactionResultEventFailure(
				context.Background(),
				fmt.Sprintf("hash-%d", nth),
				fmt.Sprintf("party-%d", i),
				fmt.Errorf("error-%d", i),
				&commandspb.LiquidityProvisionSubmission{
					MarketId:         fmt.Sprintf("market-%d", i),
					CommitmentAmount: "100",
					Fee:              "1",
					Reference:        fmt.Sprintf("lp-%d", i),
				},
			),
		)
	}

	return out
}
