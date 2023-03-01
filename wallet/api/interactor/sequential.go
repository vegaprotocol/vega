package interactor

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/wallet/api"
)

var ErrRequestAlreadyBeingProcessed = errors.New("a request is already being processed")

// SequentialInteractor is built to handle one request at a time.
// Concurrent requests are not supported and will result in errors.
type SequentialInteractor struct {
	// userCtx is the context used to listen to the user-side cancellation
	// requests. It interrupts the wait on responses.
	userCtx context.Context

	receptionChan chan<- Interaction
	responseChan  <-chan Interaction

	isProcessingRequest atomic.Bool
}

func (i *SequentialInteractor) NotifyInteractionSessionBegan(_ context.Context, traceID string) error {
	// We reject all incoming request as long as there is a request being
	// processed.
	if !i.isProcessingRequest.CompareAndSwap(false, true) {
		return ErrRequestAlreadyBeingProcessed
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    InteractionSessionBeganName,
		Data:    InteractionSessionBegan{},
	}

	return nil
}

func (i *SequentialInteractor) NotifyInteractionSessionEnded(_ context.Context, traceID string) {
	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    InteractionSessionEndedName,
		Data:    InteractionSessionEnded{},
	}

	i.isProcessingRequest.Swap(false)
}

func (i *SequentialInteractor) NotifyError(ctx context.Context, traceID string, t api.ErrorType, err error) {
	if err := ctx.Err(); err != nil {
		return
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    ErrorOccurredName,
		Data: ErrorOccurred{
			Type:  string(t),
			Error: err.Error(),
		},
	}
}

func (i *SequentialInteractor) NotifySuccessfulTransaction(ctx context.Context, traceID, txHash, deserializedInputData, tx string, sentAt time.Time, host string) {
	if err := ctx.Err(); err != nil {
		return
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    TransactionSucceededName,
		Data: TransactionSucceeded{
			DeserializedInputData: deserializedInputData,
			TxHash:                txHash,
			Tx:                    tx,
			SentAt:                sentAt,
			Node: SelectedNode{
				Host: host,
			},
		},
	}
}

func (i *SequentialInteractor) NotifyFailedTransaction(ctx context.Context, traceID, deserializedInputData, tx string, err error, sentAt time.Time, host string) {
	if err := ctx.Err(); err != nil {
		return
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    TransactionFailedName,
		Data: TransactionFailed{
			DeserializedInputData: deserializedInputData,
			Tx:                    tx,
			Error:                 err,
			SentAt:                sentAt,
			Node: SelectedNode{
				Host: host,
			},
		},
	}
}

func (i *SequentialInteractor) NotifySuccessfulRequest(ctx context.Context, traceID string, message string) {
	if err := ctx.Err(); err != nil {
		return
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestSucceededName,
		Data: RequestSucceeded{
			Message: message,
		},
	}
}

func (i *SequentialInteractor) Log(ctx context.Context, traceID string, t api.LogType, msg string) {
	if err := ctx.Err(); err != nil {
		return
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    LogName,
		Data: Log{
			Type:    string(t),
			Message: msg,
		},
	}
}

func (i *SequentialInteractor) RequestWalletConnectionReview(ctx context.Context, traceID, hostname string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", api.ErrRequestInterrupted
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestWalletConnectionReviewName,
		Data: RequestWalletConnectionReview{
			Hostname: hostname,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, WalletConnectionDecisionName)
	if err != nil {
		return "", err
	}

	decision, ok := interaction.Data.(WalletConnectionDecision)
	if !ok {
		return "", InvalidResponsePayloadError(WalletConnectionDecisionName)
	}

	return decision.ConnectionApproval, nil
}

func (i *SequentialInteractor) RequestWalletSelection(ctx context.Context, traceID, hostname string, availableWallets []string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", api.ErrRequestInterrupted
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestWalletSelectionName,
		Data: RequestWalletSelection{
			Hostname:         hostname,
			AvailableWallets: availableWallets,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, SelectedWalletName)
	if err != nil {
		return "", err
	}

	selectedWallet, ok := interaction.Data.(SelectedWallet)
	if !ok {
		return "", InvalidResponsePayloadError(SelectedWalletName)
	}

	return selectedWallet.Wallet, nil
}

func (i *SequentialInteractor) RequestPassphrase(ctx context.Context, traceID, wallet string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", api.ErrRequestInterrupted
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestPassphraseName,
		Data: RequestPassphrase{
			Wallet: wallet,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, EnteredPassphraseName)
	if err != nil {
		return "", err
	}

	enteredPassphrase, ok := interaction.Data.(EnteredPassphrase)
	if !ok {
		return "", InvalidResponsePayloadError(EnteredPassphraseName)
	}
	return enteredPassphrase.Passphrase, nil
}

func (i *SequentialInteractor) RequestPermissionsReview(ctx context.Context, traceID, hostname, wallet string, perms map[string]string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, api.ErrRequestInterrupted
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestPermissionsReviewName,
		Data: RequestPermissionsReview{
			Hostname:    hostname,
			Wallet:      wallet,
			Permissions: perms,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, DecisionName)
	if err != nil {
		return false, err
	}

	approval, ok := interaction.Data.(Decision)
	if !ok {
		return false, InvalidResponsePayloadError(DecisionName)
	}
	return approval.Approved, nil
}

func (i *SequentialInteractor) RequestTransactionReviewForSending(ctx context.Context, traceID, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, api.ErrRequestInterrupted
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestTransactionReviewForSendingName,
		Data: RequestTransactionReviewForSending{
			Hostname:    hostname,
			Wallet:      wallet,
			PublicKey:   pubKey,
			Transaction: transaction,
			ReceivedAt:  receivedAt,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, DecisionName)
	if err != nil {
		return false, err
	}

	approval, ok := interaction.Data.(Decision)
	if !ok {
		return false, InvalidResponsePayloadError(DecisionName)
	}
	return approval.Approved, nil
}

func (i *SequentialInteractor) RequestTransactionReviewForSigning(ctx context.Context, traceID, hostname, wallet, pubKey, transaction string, receivedAt time.Time) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, api.ErrRequestInterrupted
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestTransactionReviewForSigningName,
		Data: RequestTransactionReviewForSigning{
			Hostname:    hostname,
			Wallet:      wallet,
			PublicKey:   pubKey,
			Transaction: transaction,
			ReceivedAt:  receivedAt,
		},
	}

	interaction, err := i.waitForResponse(ctx, traceID, DecisionName)
	if err != nil {
		return false, err
	}

	approval, ok := interaction.Data.(Decision)
	if !ok {
		return false, InvalidResponsePayloadError(DecisionName)
	}
	return approval.Approved, nil
}

func (i *SequentialInteractor) waitForResponse(ctx context.Context, traceID string, expectedResponseName InteractionName) (Interaction, error) {
	var response Interaction
	running := true
	for running {
		select {
		case <-ctx.Done():
			return Interaction{}, api.ErrRequestInterrupted
		case <-i.userCtx.Done():
			return Interaction{}, api.ErrUserCloseTheConnection
		case r := <-i.responseChan:
			response = r
			running = false
		}
	}

	if response.TraceID != traceID {
		return Interaction{}, TraceIDMismatchError(traceID, response.TraceID)
	}

	if response.Name == CancelRequestName {
		return Interaction{}, api.ErrUserCancelledTheRequest
	}

	if response.Name != expectedResponseName {
		return Interaction{}, WrongResponseTypeError(expectedResponseName, response.Name)
	}

	return response, nil
}

func NewSequentialInteractor(userCtx context.Context, receptionChan chan<- Interaction, responseChan <-chan Interaction) *SequentialInteractor {
	i := &SequentialInteractor{
		userCtx:             userCtx,
		receptionChan:       receptionChan,
		responseChan:        responseChan,
		isProcessingRequest: atomic.Bool{},
	}

	i.isProcessingRequest.Swap(false)

	return i
}
