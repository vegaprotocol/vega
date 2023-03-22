package interactor_test

import (
	"context"
	"sync"
	"testing"
	"time"

	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api/interactor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParallelInteractor(t *testing.T) {
	t.Run("Request wallet connection review succeeds", testRequestWalletConnectionReviewSucceeds)
	t.Run("Request wallet selection review succeeds", testRequestWalletSelectionSucceeds)
	t.Run("Request passphrase succeeds", testRequestPassphraseSucceeds)
	t.Run("Request permissions review succeeds", testRequestPermissionsReviewSucceeds)
	t.Run("Request transaction review for sending succeeds", testRequestTransactionReviewForSendingSucceeds)
	t.Run("Request transaction review for signing succeeds", testRequestTransactionReviewForSigningSucceeds)
	t.Run("Request transaction review for checking succeeds", testRequestTransactionReviewForCheckingSucceeds)
}

func testRequestWalletConnectionReviewSucceeds(t *testing.T) {
	interactorCtx := context.Background()
	interactionCtx := context.Background()
	approval := vgrand.RandomStr(5)
	traceID := vgrand.RandomStr(4)
	hostname := vgrand.RandomStr(4)

	outboundCh := make(chan interactor.Interaction)
	defer close(outboundCh)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		receivedInteraction := <-outboundCh

		assert.Equal(t, traceID, receivedInteraction.TraceID)
		assert.Equal(t, interactor.RequestWalletConnectionReviewName, receivedInteraction.Name)
		request, ok := receivedInteraction.Data.(interactor.RequestWalletConnectionReview)
		require.True(t, ok)
		assert.Equal(t, hostname, request.Hostname)
		assert.Equal(t, uint8(1), request.StepNumber)

		request.ResponseCh <- interactor.Interaction{
			TraceID: traceID,
			Name:    interactor.WalletConnectionDecisionName,
			Data: interactor.WalletConnectionDecision{
				ConnectionApproval: approval,
			},
		}

		wg.Done()
	}()

	pInteractor := interactor.NewParallelInteractor(interactorCtx, outboundCh)
	result, err := pInteractor.RequestWalletConnectionReview(interactionCtx, traceID, 1, hostname)

	wg.Wait()

	require.NoError(t, err)
	assert.Equal(t, approval, result)
}

func testRequestWalletSelectionSucceeds(t *testing.T) {
	interactorCtx := context.Background()
	interactionCtx := context.Background()
	selectedWallet := vgrand.RandomStr(5)
	traceID := vgrand.RandomStr(4)
	hostname := vgrand.RandomStr(4)
	availableWallets := []string{
		vgrand.RandomStr(4),
		vgrand.RandomStr(4),
	}

	outboundCh := make(chan interactor.Interaction)
	defer close(outboundCh)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		receivedInteraction := <-outboundCh

		assert.Equal(t, traceID, receivedInteraction.TraceID)
		assert.Equal(t, interactor.RequestWalletSelectionName, receivedInteraction.Name)
		request, ok := receivedInteraction.Data.(interactor.RequestWalletSelection)
		require.True(t, ok)
		assert.Equal(t, hostname, request.Hostname)
		assert.Equal(t, uint8(1), request.StepNumber)

		request.ResponseCh <- interactor.Interaction{
			TraceID: traceID,
			Name:    interactor.SelectedWalletName,
			Data: interactor.SelectedWallet{
				Wallet: selectedWallet,
			},
		}

		wg.Done()
	}()

	pInteractor := interactor.NewParallelInteractor(interactorCtx, outboundCh)
	result, err := pInteractor.RequestWalletSelection(interactionCtx, traceID, 1, hostname, availableWallets)

	wg.Wait()

	require.NoError(t, err)
	assert.Equal(t, selectedWallet, result)
}

func testRequestPassphraseSucceeds(t *testing.T) {
	interactorCtx := context.Background()
	interactionCtx := context.Background()
	passphrase := vgrand.RandomStr(5)
	traceID := vgrand.RandomStr(4)
	wallet := vgrand.RandomStr(4)
	reason := vgrand.RandomStr(4)

	outboundCh := make(chan interactor.Interaction)
	defer close(outboundCh)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		receivedInteraction := <-outboundCh

		assert.Equal(t, traceID, receivedInteraction.TraceID)
		assert.Equal(t, interactor.RequestPassphraseName, receivedInteraction.Name)
		request, ok := receivedInteraction.Data.(interactor.RequestPassphrase)
		require.True(t, ok)
		assert.Equal(t, wallet, request.Wallet)
		assert.Equal(t, reason, request.Reason)
		assert.Equal(t, uint8(1), request.StepNumber)

		request.ResponseCh <- interactor.Interaction{
			TraceID: traceID,
			Name:    interactor.EnteredPassphraseName,
			Data: interactor.EnteredPassphrase{
				Passphrase: passphrase,
			},
		}

		wg.Done()
	}()

	pInteractor := interactor.NewParallelInteractor(interactorCtx, outboundCh)
	result, err := pInteractor.RequestPassphrase(interactionCtx, traceID, 1, wallet, reason)

	wg.Wait()

	require.NoError(t, err)
	assert.Equal(t, passphrase, result)
}

func testRequestPermissionsReviewSucceeds(t *testing.T) {
	interactorCtx := context.Background()
	interactionCtx := context.Background()
	traceID := vgrand.RandomStr(4)
	wallet := vgrand.RandomStr(4)
	hostname := vgrand.RandomStr(4)
	perms := map[string]string{
		vgrand.RandomStr(4): vgrand.RandomStr(4),
	}

	outboundCh := make(chan interactor.Interaction)
	defer close(outboundCh)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		receivedInteraction := <-outboundCh

		assert.Equal(t, traceID, receivedInteraction.TraceID)
		assert.Equal(t, interactor.RequestPermissionsReviewName, receivedInteraction.Name)
		request, ok := receivedInteraction.Data.(interactor.RequestPermissionsReview)
		require.True(t, ok)
		assert.Equal(t, wallet, request.Wallet)
		assert.Equal(t, hostname, request.Hostname)
		assert.Equal(t, perms, request.Permissions)
		assert.Equal(t, uint8(1), request.StepNumber)

		request.ResponseCh <- interactor.Interaction{
			TraceID: traceID,
			Name:    interactor.DecisionName,
			Data: interactor.Decision{
				Approved: true,
			},
		}

		wg.Done()
	}()

	pInteractor := interactor.NewParallelInteractor(interactorCtx, outboundCh)
	result, err := pInteractor.RequestPermissionsReview(interactionCtx, traceID, 1, hostname, wallet, perms)

	wg.Wait()

	require.NoError(t, err)
	assert.True(t, result)
}

func testRequestTransactionReviewForSendingSucceeds(t *testing.T) {
	interactorCtx := context.Background()
	interactionCtx := context.Background()
	traceID := vgrand.RandomStr(4)
	hostname := vgrand.RandomStr(4)
	wallet := vgrand.RandomStr(4)
	publicKey := vgrand.RandomStr(4)
	receivedAt := time.Now()
	transaction := vgrand.RandomStr(4)

	outboundCh := make(chan interactor.Interaction)
	defer close(outboundCh)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		receivedInteraction := <-outboundCh

		assert.Equal(t, traceID, receivedInteraction.TraceID)
		assert.Equal(t, interactor.RequestTransactionReviewForSendingName, receivedInteraction.Name)
		request, ok := receivedInteraction.Data.(interactor.RequestTransactionReviewForSending)
		require.True(t, ok)
		assert.Equal(t, hostname, request.Hostname)
		assert.Equal(t, wallet, request.Wallet)
		assert.Equal(t, publicKey, request.PublicKey)
		assert.Equal(t, transaction, request.Transaction)
		assert.Equal(t, receivedAt, request.ReceivedAt)
		assert.Equal(t, uint8(1), request.StepNumber)

		request.ResponseCh <- interactor.Interaction{
			TraceID: traceID,
			Name:    interactor.DecisionName,
			Data: interactor.Decision{
				Approved: true,
			},
		}

		wg.Done()
	}()

	pInteractor := interactor.NewParallelInteractor(interactorCtx, outboundCh)
	result, err := pInteractor.RequestTransactionReviewForSending(interactionCtx, traceID, 1, hostname, wallet, publicKey, transaction, receivedAt)

	wg.Wait()

	require.NoError(t, err)
	assert.True(t, result)
}

func testRequestTransactionReviewForSigningSucceeds(t *testing.T) {
	interactorCtx := context.Background()
	interactionCtx := context.Background()
	traceID := vgrand.RandomStr(4)
	hostname := vgrand.RandomStr(4)
	wallet := vgrand.RandomStr(4)
	publicKey := vgrand.RandomStr(4)
	receivedAt := time.Now()
	transaction := vgrand.RandomStr(4)

	outboundCh := make(chan interactor.Interaction)
	defer close(outboundCh)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		receivedInteraction := <-outboundCh

		assert.Equal(t, traceID, receivedInteraction.TraceID)
		assert.Equal(t, interactor.RequestTransactionReviewForSigningName, receivedInteraction.Name)
		request, ok := receivedInteraction.Data.(interactor.RequestTransactionReviewForSigning)
		require.True(t, ok)
		assert.Equal(t, hostname, request.Hostname)
		assert.Equal(t, wallet, request.Wallet)
		assert.Equal(t, publicKey, request.PublicKey)
		assert.Equal(t, transaction, request.Transaction)
		assert.Equal(t, receivedAt, request.ReceivedAt)
		assert.Equal(t, uint8(1), request.StepNumber)

		request.ResponseCh <- interactor.Interaction{
			TraceID: traceID,
			Name:    interactor.DecisionName,
			Data: interactor.Decision{
				Approved: true,
			},
		}

		wg.Done()
	}()

	pInteractor := interactor.NewParallelInteractor(interactorCtx, outboundCh)
	result, err := pInteractor.RequestTransactionReviewForSigning(interactionCtx, traceID, 1, hostname, wallet, publicKey, transaction, receivedAt)

	wg.Wait()

	require.NoError(t, err)
	assert.True(t, result)
}

func testRequestTransactionReviewForCheckingSucceeds(t *testing.T) {
	interactorCtx := context.Background()
	interactionCtx := context.Background()
	traceID := vgrand.RandomStr(4)
	hostname := vgrand.RandomStr(4)
	wallet := vgrand.RandomStr(4)
	publicKey := vgrand.RandomStr(4)
	receivedAt := time.Now()
	transaction := vgrand.RandomStr(4)

	outboundCh := make(chan interactor.Interaction)
	defer close(outboundCh)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		receivedInteraction := <-outboundCh

		assert.Equal(t, traceID, receivedInteraction.TraceID)
		assert.Equal(t, interactor.RequestTransactionReviewForCheckingName, receivedInteraction.Name)
		request, ok := receivedInteraction.Data.(interactor.RequestTransactionReviewForChecking)
		require.True(t, ok)
		assert.Equal(t, hostname, request.Hostname)
		assert.Equal(t, wallet, request.Wallet)
		assert.Equal(t, publicKey, request.PublicKey)
		assert.Equal(t, transaction, request.Transaction)
		assert.Equal(t, receivedAt, request.ReceivedAt)
		assert.Equal(t, uint8(1), request.StepNumber)

		request.ResponseCh <- interactor.Interaction{
			TraceID: traceID,
			Name:    interactor.DecisionName,
			Data: interactor.Decision{
				Approved: true,
			},
		}

		wg.Done()
	}()

	pInteractor := interactor.NewParallelInteractor(interactorCtx, outboundCh)
	result, err := pInteractor.RequestTransactionReviewForChecking(interactionCtx, traceID, 1, hostname, wallet, publicKey, transaction, receivedAt)

	wg.Wait()

	require.NoError(t, err)
	assert.True(t, result)
}
