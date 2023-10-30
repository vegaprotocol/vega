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

package v1_test

import (
	"context"
	"sync"
	"testing"
	"time"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	walletpb "code.vegaprotocol.io/vega/protos/vega/wallet/v1"
	"code.vegaprotocol.io/vega/wallet/service/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplicitConsentPolicy(t *testing.T) {
	t.Run("Requesting explicit consent succeeds", testRequestingExplicitConsentSucceeds)
	t.Run("Canceling consent requests succeeds", testCancelingConsentRequestSucceeds)
	t.Run("Reporting sent transaction succeeds", testReportingSentTransactionSucceeds)
}

func testRequestingExplicitConsentSucceeds(t *testing.T) {
	// given
	txn := &walletpb.SubmitTransactionRequest{}
	txID := vgrand.RandomStr(5)
	consentRequestsChan := make(chan v1.ConsentRequest, 1)
	sentTransactionsChan := make(chan v1.SentTransaction, 1)

	// setup
	p := v1.NewExplicitConsentPolicy(context.Background(), consentRequestsChan, sentTransactionsChan)

	go func() {
		req := <-consentRequestsChan
		d := v1.ConsentConfirmation{TxID: txID, Decision: false}
		req.Confirmation <- d
	}()

	// when
	answer, err := p.Ask(txn, txID, time.Now())
	require.Nil(t, err)
	require.False(t, answer)
}

func testCancelingConsentRequestSucceeds(t *testing.T) {
	// given
	ctx, cancelFn := context.WithCancel(context.Background())
	txn := &walletpb.SubmitTransactionRequest{}
	txID := vgrand.RandomStr(5)
	// Channels have a smaller buffer than the number of requests, on purpose.
	// We have to ensure channels are not blocking and preventing interruption
	// when full.
	consentRequestsChan := make(chan v1.ConsentRequest, 1)
	sentTransactionsChan := make(chan v1.SentTransaction, 1)

	// setup
	p := v1.NewExplicitConsentPolicy(ctx, consentRequestsChan, sentTransactionsChan)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			answer, err := p.Ask(txn, txID, time.Now())
			require.ErrorIs(t, err, v1.ErrInterruptedConsentRequest)
			assert.False(t, answer)
		}()
	}

	// interrupting the consent requests
	cancelFn()

	// waiting for all consent request to be interrupted
	wg.Wait()
}

func testReportingSentTransactionSucceeds(t *testing.T) {
	txID := vgrand.RandomStr(5)
	txHash := vgrand.RandomStr(5)
	consentRequestsChan := make(chan v1.ConsentRequest, 1)
	sentTransactionsChan := make(chan v1.SentTransaction, 1)

	// setup
	p := v1.NewExplicitConsentPolicy(context.Background(), consentRequestsChan, sentTransactionsChan)

	// when
	p.Report(v1.SentTransaction{
		TxHash: txHash,
		TxID:   txID,
		Tx:     &commandspb.Transaction{},
	})

	// then
	sentTransaction := <-sentTransactionsChan
	require.Equal(t, txHash, sentTransaction.TxHash)
	require.Equal(t, txID, sentTransaction.TxID)
}
