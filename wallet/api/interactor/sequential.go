package interactor

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/wallet/api"
)

var (
	ErrTraceIDMismatch              = errors.New("the trace IDs between request and response mismatch")
	ErrWrongResponseType            = errors.New("the received response does not match the expected response type")
	ErrRequestAlreadyBeingProcessed = errors.New("a request is already being processed")
)

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

func (i *SequentialInteractor) NotifySuccessfulTransaction(ctx context.Context, traceID, txHash, deserializedInputData, tx string, sentAt time.Time) {
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
		},
	}
}

func (i *SequentialInteractor) NotifyFailedTransaction(ctx context.Context, traceID, deserializedInputData, tx string, err error, sentAt time.Time) {
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
		},
	}
}

func (i *SequentialInteractor) NotifySuccessfulRequest(ctx context.Context, traceID string) {
	if err := ctx.Err(); err != nil {
		return
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestSucceededName,
		Data:    RequestSucceeded{},
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

	for {
		select {
		case <-ctx.Done():
			return "", api.ErrRequestInterrupted
		case <-i.userCtx.Done():
			return "", api.ErrUserCloseTheConnection
		case response := <-i.responseChan:
			if response.TraceID != traceID {
				return "", ErrTraceIDMismatch
			}
			decision, ok := response.Data.(WalletConnectionDecision)
			if !ok {
				return "", ErrWrongResponseType
			}
			return decision.ConnectionApproval, nil
		}
	}
}

func (i *SequentialInteractor) RequestWalletSelection(ctx context.Context, traceID, hostname string, availableWallets []string) (api.SelectedWallet, error) {
	if err := ctx.Err(); err != nil {
		return api.SelectedWallet{}, api.ErrRequestInterrupted
	}

	i.receptionChan <- Interaction{
		TraceID: traceID,
		Name:    RequestWalletSelectionName,
		Data: RequestWalletSelection{
			Hostname:         hostname,
			AvailableWallets: availableWallets,
		},
	}

	for {
		select {
		case <-ctx.Done():
			return api.SelectedWallet{}, api.ErrRequestInterrupted
		case <-i.userCtx.Done():
			return api.SelectedWallet{}, api.ErrUserCloseTheConnection
		case response := <-i.responseChan:
			if response.TraceID != traceID {
				return api.SelectedWallet{}, ErrTraceIDMismatch
			}
			selectedWallet, ok := response.Data.(SelectedWallet)
			if !ok {
				return api.SelectedWallet{}, ErrWrongResponseType
			}

			return api.SelectedWallet{
				Wallet:     selectedWallet.Wallet,
				Passphrase: selectedWallet.Passphrase,
			}, nil
		}
	}
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

	for {
		select {
		case <-ctx.Done():
			return "", api.ErrRequestInterrupted
		case <-i.userCtx.Done():
			return "", api.ErrUserCloseTheConnection
		case response := <-i.responseChan:
			if response.TraceID != traceID {
				return "", ErrTraceIDMismatch
			}
			enteredPassphrase, ok := response.Data.(EnteredPassphrase)
			if !ok {
				return "", ErrWrongResponseType
			}
			return enteredPassphrase.Passphrase, nil
		}
	}
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

	for {
		select {
		case <-ctx.Done():
			return false, api.ErrRequestInterrupted
		case <-i.userCtx.Done():
			return false, api.ErrUserCloseTheConnection
		case response := <-i.responseChan:
			if response.TraceID != traceID {
				return false, ErrTraceIDMismatch
			}
			decision, ok := response.Data.(Decision)
			if !ok {
				return false, ErrWrongResponseType
			}
			return decision.Approved, nil
		}
	}
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

	for {
		select {
		case <-ctx.Done():
			return false, api.ErrRequestInterrupted
		case <-i.userCtx.Done():
			return false, api.ErrUserCloseTheConnection
		case response := <-i.responseChan:
			if response.TraceID != traceID {
				return false, ErrTraceIDMismatch
			}
			approval, ok := response.Data.(Decision)
			if !ok {
				return false, ErrWrongResponseType
			}
			return approval.Approved, nil
		}
	}
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

	for {
		select {
		case <-ctx.Done():
			return false, api.ErrRequestInterrupted
		case <-i.userCtx.Done():
			return false, api.ErrUserCloseTheConnection
		case response := <-i.responseChan:
			if response.TraceID != traceID {
				return false, ErrTraceIDMismatch
			}
			approval, ok := response.Data.(Decision)
			if !ok {
				return false, ErrWrongResponseType
			}
			return approval.Approved, nil
		}
	}
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
